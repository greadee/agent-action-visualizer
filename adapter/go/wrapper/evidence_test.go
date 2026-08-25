package wrapper

import (
	"context"
	"sync"
	"testing"
	"time"

	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

func TestEvidenceGatePrefersStructuredEvidenceOverFallback(t *testing.T) {
	target := &recordingEmitter{}
	gate := newEvidenceGate(target, 8, 30*time.Millisecond)
	fallback := evidenceTestEvent("fallback", protocol.SourceFilesystem, protocol.ConfidenceObserved, protocol.EventFileModified)
	structured := evidenceTestEvent("structured", protocol.SourceStructuredStream, protocol.ConfidenceExact, protocol.EventFilePatched)

	gate.Emit(context.Background(), []protocol.Event{fallback})
	gate.Emit(context.Background(), []protocol.Event{structured})
	gate.stop(100 * time.Millisecond)

	events := target.events()
	if len(events) != 1 || events[0].EventID != "structured" {
		t.Fatalf("events = %#v", events)
	}
}

func TestEvidenceGateSuppressesLateFallbackDuplicate(t *testing.T) {
	target := &recordingEmitter{}
	gate := newEvidenceGate(target, 8, 40*time.Millisecond)
	structured := evidenceTestEvent("structured", protocol.SourceNativeHook, protocol.ConfidenceExact, protocol.EventFileModified)
	fallback := evidenceTestEvent("fallback", protocol.SourceFilesystem, protocol.ConfidenceObserved, protocol.EventFilePatched)

	gate.Emit(context.Background(), []protocol.Event{structured})
	time.Sleep(5 * time.Millisecond)
	gate.Emit(context.Background(), []protocol.Event{fallback})
	gate.stop(100 * time.Millisecond)

	events := target.events()
	if len(events) != 1 || events[0].EventID != "structured" {
		t.Fatalf("events = %#v", events)
	}
}

func TestEvidenceGateFlushesUnmatchedFallback(t *testing.T) {
	target := &recordingEmitter{}
	gate := newEvidenceGate(target, 8, 15*time.Millisecond)
	fallback := evidenceTestEvent("fallback", protocol.SourceFilesystem, protocol.ConfidenceObserved, protocol.EventFileCreated)
	gate.Emit(context.Background(), []protocol.Event{fallback})

	deadline := time.Now().Add(250 * time.Millisecond)
	for len(target.events()) == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	gate.stop(100 * time.Millisecond)
	events := target.events()
	if len(events) != 1 || events[0].EventID != "fallback" {
		t.Fatalf("events = %#v", events)
	}
}

func evidenceTestEvent(id string, source protocol.SourceType, confidence protocol.Confidence, eventType protocol.EventType) protocol.Event {
	return protocol.Event{
		SchemaVersion:    protocol.SchemaVersion,
		EventID:          id,
		SessionID:        "session",
		ProjectRoot:      `C:\workspace`,
		Path:             `C:\workspace\file.go`,
		SourceType:       source,
		SourceConfidence: confidence,
		EventType:        eventType,
		Timestamp:        time.Unix(1, 0).UTC(),
	}
}

type recordingEmitter struct {
	mu     sync.Mutex
	values []protocol.Event
}

func (e *recordingEmitter) Emit(_ context.Context, events []protocol.Event) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.values = append(e.values, events...)
}

func (e *recordingEmitter) events() []protocol.Event {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]protocol.Event(nil), e.values...)
}
