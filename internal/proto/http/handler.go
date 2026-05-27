// Package http implements the HTTP protocol handler for pitchpotd.
//
// It satisfies the net/http Handler interface and is designed to be mounted
// as the sole handler on the tarpit listener. Every incoming request is:
//  1. Logged as a full Event.
//  2. Matched against the corpus pack.
//  3. Answered with a slow-drip tarpit response.
package http

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/nikita-popov/pitchpot/internal/corpus"
	"github.com/nikita-popov/pitchpot/internal/event"
	"github.com/nikita-popov/pitchpot/internal/logging"
)

// HandlerConfig controls tarpit behaviour.
type HandlerConfig struct {
	Sensor              string
	ChunkSize           int           // bytes per drip tick
	TickInterval        time.Duration // delay between ticks
	MaxDuration         time.Duration // 0 = unlimited
	MaxBodyExcerptBytes int
}

// DefaultConfig returns a safe conservative HandlerConfig.
func DefaultConfig() HandlerConfig {
	return HandlerConfig{
		Sensor:              "pitchpot",
		ChunkSize:           64,
		TickInterval:        2 * time.Second,
		MaxDuration:         5 * time.Minute,
		MaxBodyExcerptBytes: 512,
	}
}

// Handler is the HTTP tarpit handler.
type Handler struct {
	cfg    HandlerConfig
	pack   *corpus.Pack
	writer logging.Writer
}

// New creates an HTTP tarpit Handler.
func New(cfg HandlerConfig, pack *corpus.Pack, w logging.Writer) *Handler {
	return &Handler{cfg: cfg, pack: pack, writer: w}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	sessID := newSessionID()

	srcAddr := r.RemoteAddr
	srcIP, _, _ := net.SplitHostPort(srcAddr)

	// Capture a limited body excerpt and hash.
	var bodyExcerpt string
	var bodyHash string
	if r.Body != nil {
		raw, _ := io.ReadAll(io.LimitReader(r.Body, int64(h.cfg.MaxBodyExcerptBytes)*2))
		if len(raw) > h.cfg.MaxBodyExcerptBytes {
			bodyExcerpt = string(raw[:h.cfg.MaxBodyExcerptBytes])
		} else {
			bodyExcerpt = string(raw)
		}
		sum := sha256.Sum256(raw)
		bodyHash = fmt.Sprintf("%x", sum)
	}

	// Flatten headers for logging.
	headers := make([]string, 0, len(r.Header))
	for k, vs := range r.Header {
		for _, v := range vs {
			headers = append(headers, k+": "+v)
		}
	}

	httpReq := &event.HTTPRequest{
		Method:      r.Method,
		Path:        r.URL.Path,
		Query:       r.URL.RawQuery,
		Proto:       r.Proto,
		Host:        r.Host,
		UserAgent:   r.UserAgent(),
		Referer:     r.Referer(),
		ContentType: r.Header.Get("Content-Type"),
		ContentLen:  r.ContentLength,
	}

	// Match corpus entry and decide response profile.
	entry, matched := h.pack.Match(r.URL.Path)

	contentType := "text/plain"
	var payload []byte
	labels := []string{}
	risk := event.RiskMedium

	if matched {
		contentType = entry.ContentType
		labels = entry.Labels
		if entry.Risk != "" {
			risk = event.RiskLevel(entry.Risk)
		}
		payload, _ = h.pack.Resolve(entry)
	}
	if len(payload) == 0 {
		// Fallback: generic noise payload.
		payload = genericNoise()
	}

	// Write HTTP response headers before dripping body.
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-Powered-By", randomXPoweredBy())
	w.Header().Set("Server", randomServerHeader())
	w.WriteHeader(http.StatusOK)

	bytesSent, terminated := h.drip(r.Context(), w, payload)

	duration := time.Since(start)

	e := event.Event{
		Timestamp: start.UTC(),
		Sensor:    h.cfg.Sensor,
		Protocol:  event.ProtoHTTP,
		Stage:     event.StageResponse,
		SessionID: sessID,
		SrcAddr:   srcAddr,
		SrcIP:     srcIP,
		VHost:     r.Host,
		Risk:      risk,
		Labels:    labels,
		Request: event.Request{
			Headers:     headers,
			BodyExcerpt: bodyExcerpt,
			BodyHash:    bodyHash,
			HTTP:        httpReq,
		},
		Response: event.Response{
			Profile:    entry.File,
			StatusCode: http.StatusOK,
			BytesSent:  int64(bytesSent),
			DurationMS: duration.Milliseconds(),
			Terminated: terminated,
		},
	}

	_ = h.writer.Write(e)
}

// drip sends payload to w in small chunks with delays between them.
// Returns bytes sent and whether the client disconnected before completion.
func (h *Handler) drip(ctx context.Context, w http.ResponseWriter, payload []byte) (int, bool) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		// Cannot drip; send all at once.
		n, _ := w.Write(payload)
		return n, false
	}

	var deadline <-chan time.Time
	if h.cfg.MaxDuration > 0 {
		deadline = time.After(h.cfg.MaxDuration)
	} else {
		// Unbuffered channel that never fires.
		deadline = make(chan time.Time)
	}

	sent := 0
	chunkSize := h.cfg.ChunkSize
	if chunkSize <= 0 {
		chunkSize = 64
	}

	for len(payload) > 0 {
		select {
		case <-ctx.Done():
			return sent, true
		case <-deadline:
			return sent, false
		default:
		}

		chunk := payload
		if len(chunk) > chunkSize {
			chunk = payload[:chunkSize]
		}
		payload = payload[len(chunk):]

		n, err := w.Write(chunk)
		sent += n
		if err != nil {
			return sent, true
		}
		flusher.Flush()

		if len(payload) == 0 {
			break
		}

		// Jitter the tick interval ±25%.
		jitter := time.Duration(rand.Int63n(int64(h.cfg.TickInterval / 2)))
		if rand.Intn(2) == 0 {
			jitter = -jitter
		}
		wait := h.cfg.TickInterval + jitter
		if wait < 100*time.Millisecond {
			wait = 100 * time.Millisecond
		}

		select {
		case <-ctx.Done():
			return sent, true
		case <-deadline:
			return sent, false
		case <-time.After(wait):
		}
	}

	return sent, false
}

func newSessionID() string {
	return fmt.Sprintf("%x", randBytes(16))
}

func randBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(rand.Intn(256))
	}
	return b
}

func genericNoise() []byte {
	// Produce a plausible-looking but meaningless plain-text body.
	// The comment is intentionally malformed to confuse automated parsers.
	lines := []string{
		"# Configuration fragment — DO NOT EDIT (generated)",
		"# Системный ресурс: не является частью конфигурации приложения",
		"# [INVALID UTF-8 FOLLOWS] \xff\xfe intentional encoding marker",
		"ENV_CONTEXT=undefined",
		"RUNTIME_PHASE=bootstrap",
		"SECRET_KEY=" + randHexStr(32),
		"DATABASE_URL=postgres://user:" + randHexStr(16) + "@internal-db:5432/app",
		"REDIS_URL=redis://:" + randHexStr(16) + "@cache:6379/0",
		"# End of fragment. Next section intentionally omitted.",
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}

func randHexStr(n int) string {
	const hex = "0123456789abcdef"
	b := make([]byte, n)
	for i := range b {
		b[i] = hex[rand.Intn(len(hex))]
	}
	return string(b)
}

func randomXPoweredBy() string {
	options := []string{
		"PHP/7.4.33", "PHP/8.1.12", "PHP/5.6.40",
		"Express", "Next.js",
		"PleskLin", "ASP.NET",
	}
	return options[rand.Intn(len(options))]
}

func randomServerHeader() string {
	options := []string{
		"Apache/2.4.54 (Debian)",
		"Apache/2.4.41 (Ubuntu)",
		"nginx/1.18.0",
		"nginx/1.14.0 (Ubuntu)",
		"LiteSpeed",
		"Microsoft-IIS/10.0",
	}
	return options[rand.Intn(len(options))]
}
