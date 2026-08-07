package protocol

import (
	"encoding/json"
	"fmt"
	"strings"
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

func TestEventValidationEnforcesSecurityBounds(t *testing.T) {
	valid := func() Event {
		return Event{SchemaVersion: SchemaVersion, EventID: "event", EventType: EventFileRead, SourceType: SourceNativeHook, SourceConfidence: ConfidenceExact, Timestamp: time.Unix(1, 0).UTC()}
	}
	tests := map[string]func(*Event){
		"overlong identifier": func(event *Event) { event.EventID = strings.Repeat("x", 129) },
		"overlong path":       func(event *Event) { event.Path = strings.Repeat("x", 4097) },
		"invalid UTF-8":       func(event *Event) { event.ActionLabel = string([]byte{0xff}) },
		"NUL label":           func(event *Event) { event.ActionLabel = "safe\x00hidden" },
		"negative process":    func(event *Event) { event.ProcessID = -1 },
		"negative monotonic":  func(event *Event) { event.MonotonicTimestamp = -1 },
		"oversized metadata": func(event *Event) {
			event.Metadata = map[string]interface{}{"data": strings.Repeat("x", MaxMetadataBytes)}
		},
		"non-JSON metadata": func(event *Event) { event.Metadata = map[string]interface{}{"data": func() {}} },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			event := valid()
			mutate(&event)
			if err := event.Validate(); err == nil {
				t.Fatal("invalid event was accepted")
			}
		})
	}

	deep := valid()
	value := map[string]interface{}{"leaf": true}
	for depth := 0; depth <= MaxMetadataDepth; depth++ {
		value = map[string]interface{}{fmt.Sprintf("level-%d", depth): value}
	}
	deep.Metadata = value
	if err := deep.Validate(); err == nil {
		t.Fatal("deeply nested metadata was accepted")
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
