// Package logging provides event writers for pitchpotd.
// Two writers are supported:
//   - JSONL: full Event struct, one JSON object per line. Used for analysis.
//   - BanLog: compact single-line format for fail2ban / CrowdSec filters.
package logging

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/nikita-popov/pitchpot/internal/event"
)

// Writer is the interface implemented by all log sinks.
type Writer interface {
	Write(e event.Event) error
	Close() error
}

// JSONLWriter writes full event structs as JSONL.
type JSONLWriter struct {
	mu sync.Mutex
	f  *os.File
	enc *json.Encoder
}

// NewJSONLWriter opens or creates the file at path and returns a JSONLWriter.
func NewJSONLWriter(path string) (*JSONLWriter, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0640)
	if err != nil {
		return nil, fmt.Errorf("jsonl writer: %w", err)
	}
	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	return &JSONLWriter{f: f, enc: enc}, nil
}

// NewJSONLWriterTo writes to any io.Writer (useful in tests).
func NewJSONLWriterTo(w io.Writer) *JSONLWriter {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return &JSONLWriter{enc: enc}
}

func (w *JSONLWriter) Write(e event.Event) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.enc.Encode(e)
}

func (w *JSONLWriter) Close() error {
	if w.f != nil {
		return w.f.Close()
	}
	return nil
}

// BanLogWriter writes a compact one-line entry per event.
// Format (tab-separated):
//
//	<timestamp> <src_ip> <protocol> <stage> <risk> [label,...] <detail>
//
// This format is designed for simple fail2ban / CrowdSec regex filters.
type BanLogWriter struct {
	mu sync.Mutex
	f  *os.File
}

// NewBanLogWriter opens or creates the compact ban log at path.
func NewBanLogWriter(path string) (*BanLogWriter, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0640)
	if err != nil {
		return nil, fmt.Errorf("banlog writer: %w", err)
	}
	return &BanLogWriter{f: f}, nil
}

func (w *BanLogWriter) Write(e event.Event) error {
	ts := e.Timestamp.UTC().Format(time.RFC3339)

	detail := ""
	if e.Request.HTTP != nil {
		detail = fmt.Sprintf("%s %s", e.Request.HTTP.Method, e.Request.HTTP.Path)
	} else if e.Request.Line != "" {
		detail = e.Request.Line
	}

	labels := ""
	if len(e.Labels) > 0 {
		for i, l := range e.Labels {
			if i > 0 {
				labels += ","
			}
			labels += l
		}
	} else {
		labels = "-"
	}

	line := fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
		ts, e.SrcIP, string(e.Protocol), string(e.Stage), string(e.Risk), labels, detail,
	)

	w.mu.Lock()
	defer w.mu.Unlock()
	_, err := w.f.WriteString(line)
	return err
}

func (w *BanLogWriter) Close() error {
	if w.f != nil {
		return w.f.Close()
	}
	return nil
}

// Multi fans out Write calls to multiple writers. Errors from individual
// writers are accumulated but do not stop delivery to remaining writers.
type Multi struct {
	writers []Writer
}

// NewMulti returns a Multi writer wrapping the provided writers.
func NewMulti(ws ...Writer) *Multi { return &Multi{writers: ws} }

func (m *Multi) Write(e event.Event) error {
	var first error
	for _, w := range m.writers {
		if err := w.Write(e); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func (m *Multi) Close() error {
	var first error
	for _, w := range m.writers {
		if err := w.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}
