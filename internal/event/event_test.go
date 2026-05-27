package event

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEventMarshalRoundtrip(t *testing.T) {
	e := Event{
		Timestamp: time.Now().UTC().Truncate(time.Second),
		Sensor:    "test-sensor",
		Protocol:  ProtoHTTP,
		Stage:     StageRequest,
		SessionID: "abc-123",
		SrcAddr:   "1.2.3.4:1234",
		SrcIP:     "1.2.3.4",
		Risk:      RiskHigh,
		Labels:    []string{"probe:env"},
		Request: Request{
			HTTP: &HTTPRequest{
				Method:    "GET",
				Path:      "/.env",
				Proto:     "HTTP/1.1",
				UserAgent: "zgrab/0.x",
			},
		},
	}

	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var e2 Event
	if err := json.Unmarshal(data, &e2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if e2.SrcIP != e.SrcIP {
		t.Errorf("SrcIP: want %s got %s", e.SrcIP, e2.SrcIP)
	}
	if e2.Request.HTTP.Path != "/.env" {
		t.Errorf("path mismatch")
	}
}
