package adapter_test

import (
	"context"
	"errors"
	"testing"
	"time"

	adapter "github.com/greadee/agent-action-visualizer/adapter/go"
	"github.com/greadee/agent-action-visualizer/adapter/go/example"
	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

func TestStaticExampleSatisfiesContract(t *testing.T) {
	event := validEvent()
	candidate := example.Static{Events: []protocol.Event{event}}
	if err := adapter.ValidateAdapter(candidate); err != nil {
		t.Fatalf("example contract: %v", err)
	}
	collector := &adapter.MockCollector{}
	candidate.Observe(context.Background(), adapter.NewBoundedEmitter(collector, adapter.EmitterConfig{}))
	events := waitForEvents(t, collector, 1)
	if events[0].EventID != event.EventID {
		t.Fatalf("event id = %q, want %q", events[0].EventID, event.EventID)
	}
}

func TestDescriptorRejectsUnsupportedAndDuplicateCapabilities(t *testing.T) {
	descriptor := adapter.Descriptor{ID: "test", Version: "1", ProtocolVersion: protocol.SchemaVersion, Capabilities: []adapter.Capability{adapter.CapabilityFileRead, adapter.CapabilityFileRead}}
	if err := descriptor.Validate(); err == nil {
		t.Fatal("expected duplicate capability rejection")
	}
	descriptor.Capabilities = []adapter.Capability{"unknown"}
	if err := descriptor.Validate(); err == nil {
		t.Fatal("expected unsupported capability rejection")
	}
}

func TestBoundedEmitterIsFailureOpenAndBounded(t *testing.T) {
	blocked := make(chan struct{})
	collector := &adapter.MockCollector{Block: blocked}
	emitter := adapter.NewBoundedEmitter(collector, adapter.EmitterConfig{MaxInFlight: 1, Deadline: 20 * time.Millisecond})
	started := time.Now()
	emitter.Emit(context.Background(), []protocol.Event{validEvent()})
	emitter.Emit(context.Background(), []protocol.Event{validEvent()})
	if elapsed := time.Since(started); elapsed > 50*time.Millisecond {
		t.Fatalf("emitter blocked observer for %s", elapsed)
	}
	time.Sleep(50 * time.Millisecond)
	if calls := collector.Calls(); calls != 0 {
		t.Fatalf("blocked collector calls = %d, want 0", calls)
	}
	close(blocked)

	failed := &adapter.MockCollector{Err: errors.New("collector unavailable")}
	adapter.NewBoundedEmitter(failed, adapter.EmitterConfig{}).Emit(context.Background(), []protocol.Event{validEvent()})
	_ = waitForEvents(t, failed, 1)
}

func TestBoundedEmitterDropsInvalidEvents(t *testing.T) {
	collector := &adapter.MockCollector{}
	emitter := adapter.NewBoundedEmitter(collector, adapter.EmitterConfig{})
	emitter.Emit(context.Background(), []protocol.Event{{EventID: "invalid"}})
	time.Sleep(25 * time.Millisecond)
	if calls := collector.Calls(); calls != 0 {
		t.Fatalf("invalid batch sent %d times", calls)
	}
}

func TestBoundedEmitterCapsBatchAndRecoversCollectorPanic(t *testing.T) {
	collector := &adapter.MockCollector{}
	emitter := adapter.NewBoundedEmitter(collector, adapter.EmitterConfig{MaxEvents: 1})
	emitter.Emit(context.Background(), []protocol.Event{validEvent(), validEvent()})
	if events := waitForEvents(t, collector, 1); len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}

	panicEmitter := adapter.NewBoundedEmitter(panicCollector{}, adapter.EmitterConfig{})
	panicEmitter.Emit(context.Background(), []protocol.Event{validEvent()})
	time.Sleep(25 * time.Millisecond)
}

func TestNormalizeProjectPathRejectsEscape(t *testing.T) {
	if _, err := adapter.NormalizeProjectPath(t.TempDir(), "../secret.txt"); err == nil {
		t.Fatal("expected path escape rejection")
	}
}

func validEvent() protocol.Event {
	return protocol.Event{
		SchemaVersion:    protocol.SchemaVersion,
		EventID:          "event-1",
		EventType:        protocol.EventFileRead,
		SourceType:       protocol.SourceNativeHook,
		SourceConfidence: protocol.ConfidenceExact,
		Timestamp:        time.Unix(1, 0).UTC(),
	}
}

func waitForEvents(t *testing.T, collector *adapter.MockCollector, count int) []protocol.Event {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if events := collector.Events(); len(events) >= count {
			return events
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d events", count)
	return nil
}

type panicCollector struct{}

func (panicCollector) Send(context.Context, []protocol.Event) error {
	panic("collector panic")
}
