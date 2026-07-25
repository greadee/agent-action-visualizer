package session

import (
	"testing"
	"time"

	workdiff "github.com/greadee/agent-action-visualizer/internal/diff"
	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

func ev(session string, kind protocol.EventType, path string, at time.Time, confidence protocol.Confidence) protocol.Event {
	return protocol.Event{SchemaVersion: protocol.SchemaVersion, EventID: string(kind) + path, SessionID: session, EventType: kind, Operation: string(kind), Path: path, Timestamp: at, SourceType: protocol.SourceNativeHook, SourceConfidence: confidence}
}

func start(t *testing.T, engine *Engine, id string, at time.Time) {
	t.Helper()
	if _, err := engine.Apply(ev(id, protocol.EventSessionStarted, "", at, protocol.ConfidenceExact)); err != nil {
		t.Fatal(err)
	}
}

func TestActivePreviousAndIntervals(t *testing.T) {
	base := time.Unix(1000, 0)
	engine := NewEngine(time.Minute)
	start(t, engine, "s", base)
	state, _ := engine.Apply(ev("s", protocol.EventFileRead, "a.go", base.Add(time.Second), protocol.ConfidenceExact))
	if state.ActivePath != "a.go" || len(state.Intervals) != 1 {
		t.Fatalf("first focus: %+v", state)
	}
	state, _ = engine.Apply(ev("s", protocol.EventFilePatched, "b.go", base.Add(11*time.Second), protocol.ConfidenceExact))
	if state.ActivePath != "b.go" || state.PreviousPath != "a.go" || state.Intervals[0].Duration != 10*time.Second {
		t.Fatalf("transition: %+v", state)
	}
	state, _ = engine.Apply(ev("s", protocol.EventSessionStopped, "", base.Add(21*time.Second), protocol.ConfidenceExact))
	if state.Intervals[1].Duration != 10*time.Second || state.StoppedAt == nil {
		t.Fatalf("stop: %+v", state)
	}
}

func TestEditOutranksReadAtSameTimestamp(t *testing.T) {
	base := time.Unix(1000, 0)
	engine := NewEngine(time.Minute)
	start(t, engine, "s", base)
	engine.Apply(ev("s", protocol.EventFilePatched, "write.go", base, protocol.ConfidenceExact))
	state, _ := engine.Apply(ev("s", protocol.EventFileRead, "read.go", base, protocol.ConfidenceExact))
	if state.ActivePath != "write.go" {
		t.Fatalf("read stole focus: %+v", state)
	}
}

func TestRenamePreservesActiveIdentity(t *testing.T) {
	base := time.Unix(1000, 0)
	engine := NewEngine(time.Minute)
	start(t, engine, "s", base)
	first := ev("s", protocol.EventFileCreated, "old.go", base, protocol.ConfidenceExact)
	first.NodeID = "node-1"
	engine.Apply(first)
	rename := ev("s", protocol.EventFileRenamed, "new.go", base.Add(time.Second), protocol.ConfidenceExact)
	rename.PreviousPath = "old.go"
	rename.NodeID = "node-1"
	state, _ := engine.Apply(rename)
	if state.ActivePath != "new.go" || state.ActiveNodeID != "node-1" || len(state.Intervals) != 1 || state.Intervals[0].Path != "new.go" {
		t.Fatalf("rename lost identity: %+v", state)
	}
}

func TestIdleTimeoutCapsDuration(t *testing.T) {
	base := time.Unix(1000, 0)
	engine := NewEngine(30 * time.Second)
	start(t, engine, "s", base)
	engine.Apply(ev("s", protocol.EventFileRead, "a.go", base, protocol.ConfidenceExact))
	state, _ := engine.Advance("s", base.Add(10*time.Minute))
	if state.Intervals[0].Duration != 30*time.Second {
		t.Fatalf("idle inflated: %s", state.Intervals[0].Duration)
	}
}

func TestSamePathAfterIdleCreatesNewAccess(t *testing.T) {
	base := time.Unix(1000, 0)
	engine := NewEngine(30 * time.Second)
	start(t, engine, "s", base)
	first := ev("s", protocol.EventFileRead, "a.go", base, protocol.ConfidenceExact)
	first.NodeID = "node-a"
	if _, err := engine.Apply(first); err != nil {
		t.Fatal(err)
	}
	if _, err := engine.Advance("s", base.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	next := ev("s", protocol.EventFilePatched, "a.go", base.Add(2*time.Minute), protocol.ConfidenceExact)
	next.NodeID = "node-a"
	state, err := engine.Apply(next)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Intervals) != 2 || state.Intervals[1].Sequence != 2 || state.Intervals[1].EndedAt != nil {
		t.Fatalf("same path did not reopen as a new access: %+v", state.Intervals)
	}
	if state.ActiveNodeID != "node-a" || state.PreviousNodeID != "" {
		t.Fatalf("same path reopen changed focus identity: %+v", state)
	}
}

func TestFocusStateCarriesOperationAndSecondaryPaths(t *testing.T) {
	base := time.Unix(1000, 0)
	engine := NewEngine(time.Minute)
	start(t, engine, "s", base)
	first := ev("s", protocol.EventFileRead, "a.go", base.Add(time.Second), protocol.ConfidenceObserved)
	first.NodeID = "node-a"
	engine.Apply(first)
	next := ev("s", protocol.EventFilePatched, "b.go", base.Add(2*time.Second), protocol.ConfidenceExact)
	next.NodeID = "node-b"
	next.Metadata = map[string]interface{}{"secondary_paths": []string{"c.go", "c.go", "a.go"}}
	state, err := engine.Apply(next)
	if err != nil {
		t.Fatal(err)
	}
	if state.ActiveNodeID != "node-b" || state.PreviousNodeID != "node-a" {
		t.Fatalf("node state not retained: %+v", state)
	}
	if state.ActiveOperation != string(protocol.EventFilePatched) || state.ActiveConfidence != protocol.ConfidenceExact || !state.ActiveTimestamp.Equal(next.Timestamp) {
		t.Fatalf("active event details not retained: %+v", state)
	}
	if len(state.SecondaryPaths) != 2 || state.SecondaryPaths[0] != "a.go" || state.SecondaryPaths[1] != "c.go" {
		t.Fatalf("secondary paths not normalized: %#v", state.SecondaryPaths)
	}
}

func TestAsynchronousWorkResultUpdatesExactInterval(t *testing.T) {
	base := time.Unix(1000, 0)
	engine := NewEngine(time.Minute)
	start(t, engine, "s", base)
	engine.Apply(ev("s", protocol.EventFilePatched, "a.go", base, protocol.ConfidenceExact))
	engine.Apply(ev("s", protocol.EventFilePatched, "b.go", base.Add(time.Second), protocol.ConfidenceExact))
	added, deleted := int64(12), int64(4)
	state, err := engine.SetWorkResult(
		"s", 1, &added, &deleted, workdiff.StatusKnown,
		workdiff.SourceStructuredPatch, protocol.ConfidenceExact,
	)
	if err != nil {
		t.Fatal(err)
	}
	if state.Intervals[0].LinesAdded == nil || *state.Intervals[0].LinesAdded != 12 ||
		state.Intervals[0].LinesDeleted == nil || *state.Intervals[0].LinesDeleted != 4 {
		t.Fatalf("work result not attached: %+v", state.Intervals)
	}
	if state.Intervals[0].WorkSource != workdiff.SourceStructuredPatch ||
		state.Intervals[1].LinesAdded != nil {
		t.Fatalf("work provenance or target incorrect: %+v", state.Intervals)
	}
}
