package protocol

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEventValidationAndRoundTrip(t *testing.T) {
	added := int64(12)
	event := Event{SchemaVersion: SchemaVersion, EventID: "evt-1", EventType: EventFilePatched, SourceType: SourceNativeHook, SourceConfidence: ConfidenceExact, Timestamp: time.Unix(1_700_000_000, 0).UTC(), Path: "internal/session/session.go", LinesAdded: &added}
	if err := event.Validate(); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}
	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Event
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Path != event.Path || decoded.LinesAdded == nil || *decoded.LinesAdded != added {
		t.Fatalf("round trip mismatch: %#v", decoded)
	}
}

func TestUnknownFieldsRemainCompatible(t *testing.T) {
	payload := []byte(`{"schema_version":"1.0","event_id":"evt-2","event_type":"file_read","source_type":"native_hook","source_confidence":"exact","timestamp":"2026-07-21T20:00:00Z","future_field":{"enabled":true}}`)
	var event Event
	if err := json.Unmarshal(payload, &event); err != nil {
		t.Fatalf("unknown field rejected: %v", err)
	}
	if err := event.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidEventsFailSafely(t *testing.T) {
	now := time.Now()
	tests := []Event{
		{SchemaVersion: "2.0", EventID: "x", EventType: EventFileRead, SourceType: SourceNativeHook, SourceConfidence: ConfidenceExact, Timestamp: now},
		{SchemaVersion: SchemaVersion, EventID: "", EventType: EventFileRead, SourceType: SourceNativeHook, SourceConfidence: ConfidenceExact, Timestamp: now},
		{SchemaVersion: SchemaVersion, EventID: "x", EventType: "invented", SourceType: SourceNativeHook, SourceConfidence: ConfidenceExact, Timestamp: now},
	}
	for i, event := range tests {
		if err := event.Validate(); err == nil {
			t.Fatalf("case %d was accepted", i)
		}
	}
}
