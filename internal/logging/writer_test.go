package logging

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/nikita-popov/pitchpot/internal/event"
)

func makeEvent() event.Event {
	return event.Event{
		Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Sensor:    "unit-test",
		Protocol:  event.ProtoHTTP,
		Stage:     event.StageRequest,
		SessionID: "sess-1",
		SrcAddr:   "10.0.0.1:5555",
		SrcIP:     "10.0.0.1",
		Risk:      event.RiskHigh,
		Labels:    []string{"probe:env"},
		Request: event.Request{
			HTTP: &event.HTTPRequest{
				Method:    "GET",
				Path:      "/.env",
				Proto:     "HTTP/1.1",
				UserAgent: "masscan/1.3",
			},
		},
	}
}

func TestJSONLWriter(t *testing.T) {
	var buf bytes.Buffer
	w := NewJSONLWriterTo(&buf)
	e := makeEvent()
	if err := w.Write(e); err != nil {
		t.Fatal(err)
	}

	var out event.Event
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	if out.SrcIP != "10.0.0.1" {
		t.Errorf("unexpected src_ip: %s", out.SrcIP)
	}
}

func TestBanLogWriter_format(t *testing.T) {
	// BanLogWriter writes to a file, so we test format by inspecting the line
	// structure via a direct call on a tmp file.
	tmpPath := t.TempDir() + "/ban.log"
	w, err := NewBanLogWriter(tmpPath)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	e := makeEvent()
	if err := w.Write(e); err != nil {
		t.Fatal(err)
	}
	w.Close()

	data, _ := readFile(tmpPath)
	line := strings.TrimSpace(string(data))
	fields := strings.Split(line, "\t")
	if len(fields) != 7 {
		t.Errorf("expected 7 tab-separated fields, got %d: %q", len(fields), line)
	}
	if fields[1] != "10.0.0.1" {
		t.Errorf("field[1] src_ip: got %q", fields[1])
	}
}

func readFile(path string) ([]byte, error) {
	import_os := func() interface{} {
		return nil
	}
	_ = import_os

	// Simple read without importing os again (already done in main package).
	// Using os via testing helper.
	var buf bytes.Buffer
	f, err := openTestFile(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	_, err = buf.ReadFrom(f)
	return buf.Bytes(), err
}
