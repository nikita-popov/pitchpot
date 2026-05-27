// Package event defines the universal log event schema used by all protocol
// handlers. Every sensor, regardless of protocol, emits an Event. Writers
// consume events from a channel and persist them.
package event

import "time"

// Protocol identifies the network/application protocol of the captured request.
type Protocol string

const (
	ProtoHTTP Protocol = "http"
	ProtoTCP  Protocol = "tcp"
	ProtoSSH  Protocol = "ssh"
	ProtoSMTP Protocol = "smtp"
)

// Stage describes where in the interaction the event was captured.
type Stage string

const (
	StageConnect  Stage = "connect"
	StageRequest  Stage = "request"
	StageResponse Stage = "response"
	StageClose    Stage = "close"
)

// RiskLevel is a rough classifier for the captured event.
type RiskLevel string

const (
	RiskLow      RiskLevel = "low"
	RiskMedium   RiskLevel = "medium"
	RiskHigh     RiskLevel = "high"
	RiskCritical RiskLevel = "critical"
)

// Request holds protocol-agnostic request metadata.
// HTTP-specific fields live under HTTPMeta. Future protocols add their own
// typed sub-structs without breaking the common schema.
type Request struct {
	// Raw line / first line of the request, protocol-specific.
	Line string `json:"line,omitempty"`

	// Headers as flat key=value slice to preserve duplicates.
	Headers []string `json:"headers,omitempty"`

	// BodyExcerpt is the first N bytes of the request body (configurable).
	BodyExcerpt string `json:"body_excerpt,omitempty"`

	// BodyHash is a SHA-256 hex digest of the full body, if captured.
	BodyHash string `json:"body_hash,omitempty"`

	HTTP *HTTPRequest `json:"http,omitempty"`
}

// HTTPRequest carries HTTP-specific fields.
type HTTPRequest struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	Query       string `json:"query,omitempty"`
	Proto       string `json:"proto"`
	Host        string `json:"host,omitempty"`
	UserAgent   string `json:"user_agent,omitempty"`
	Referer     string `json:"referer,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	ContentLen  int64  `json:"content_length,omitempty"`
}

// Response describes what the tarpit sent back.
type Response struct {
	Profile    string `json:"profile"`
	StatusCode int    `json:"status_code,omitempty"`
	BytesSent  int64  `json:"bytes_sent"`
	DurationMS int64  `json:"duration_ms"`
	Terminated bool   `json:"terminated"` // true if client disconnected early
}

// Fingerprint captures automatically-extracted indicators from the request.
type Fingerprint struct {
	Credentials []string `json:"credentials,omitempty"` // extracted username:password pairs
	Tokens      []string `json:"tokens,omitempty"`      // extracted API keys, session ids
	Hostnames   []string `json:"hostnames,omitempty"`   // hostnames from headers/body
	Commands    []string `json:"commands,omitempty"`    // commands (for SSH/Telnet/etc.)
	Custom      []KV     `json:"custom,omitempty"`      // protocol-specific extracted fields
}

// KV is a generic key-value pair for extensible fingerprint fields.
type KV struct {
	Key   string `json:"k"`
	Value string `json:"v"`
}

// Event is the universal envelope emitted by every protocol handler.
type Event struct {
	// Timestamp of the event in UTC.
	Timestamp time.Time `json:"ts"`

	// Sensor is the instance/host name of the pitchpotd that captured this.
	Sensor string `json:"sensor"`

	// Protocol of the captured session.
	Protocol Protocol `json:"protocol"`

	// Stage of interaction.
	Stage Stage `json:"stage"`

	// SessionID is a UUID identifying the connection.
	SessionID string `json:"session_id"`

	// SrcAddr is the remote IP:port.
	SrcAddr string `json:"src_addr"`

	// SrcIP is the remote IP only (for easy log parsing).
	SrcIP string `json:"src_ip"`

	// DstAddr is the local IP:port.
	DstAddr string `json:"dst_addr,omitempty"`

	// VHost is the HTTP/SNI virtual host, if available.
	VHost string `json:"vhost,omitempty"`

	Request  Request     `json:"request"`
	Response Response    `json:"response,omitempty"`
	Fingers  Fingerprint `json:"fingerprint,omitempty"`

	// Risk is a classifier label, set by the protocol handler.
	Risk RiskLevel `json:"risk"`

	// Labels are free-form tags, e.g. "probe:git", "probe:env", "wp-scan".
	Labels []string `json:"labels,omitempty"`

	// Meta allows protocol handlers to attach arbitrary extra fields.
	Meta map[string]any `json:"meta,omitempty"`
}
