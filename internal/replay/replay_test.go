package replay

import (
	"reflect"
	"testing"
	"time"

	workdiff "github.com/greadee/agent-action-visualizer/internal/diff"
	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

func TestReconstructIsDeterministicAndClosesAtCursor(t *testing.T) {
	base := time.Unix(1_000, 0).UTC()
	events := []protocol.Event{
		event("start", protocol.EventSessionStarted, "", base),
		event("read", protocol.EventFileRead, "a.go", base.Add(time.Second)),
		event("patch", protocol.EventFilePatched, "b.go", base.Add(4*time.Second)),
	}
	first, err := Reconstruct(events, 2, 2*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Reconstruct(events, 2, 2*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if first.Cursor != 2 || first.CursorAt != base.Add(4*time.Second) || !reflect.DeepEqual(first.State, second.State) {
		t.Fatalf("non-deterministic replay: %#v %#v", first, second)
	}
	if len(first.State.Intervals) != 2 || first.State.Intervals[1].EndedAt == nil || first.State.Intervals[1].Duration != 0 {
		t.Fatalf("cursor did not freeze active access: %#v", first.State.Intervals)
	}
}

func TestReconstructAppliesPersistedWorkResult(t *testing.T) {
	base := time.Unix(2_000, 0).UTC()
	added, deleted := int64(9), int64(4)
	events := []protocol.Event{
		event("start", protocol.EventSessionStarted, "", base),
		event("patch", protocol.EventFilePatched, "a.go", base.Add(time.Second)),
		{SchemaVersion: protocol.SchemaVersion, EventID: "diff", SessionID: "s", SourceType: protocol.SourceGit, SourceConfidence: protocol.ConfidenceExact, EventType: protocol.EventDiffCalculated, Timestamp: base.Add(2 * time.Second), LinesAdded: &added, LinesDeleted: &deleted, Status: string(workdiff.StatusKnown), Operation: string(workdiff.SourceStructuredPatch), Metadata: map[string]interface{}{"access_sequence": 1}},
	}
	snapshot, err := Reconstruct(events, 2, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	access := snapshot.State.Intervals[0]
	if access.WorkStatus != workdiff.StatusKnown || access.LinesAdded == nil || *access.LinesAdded != 9 || access.LinesDeleted == nil || *access.LinesDeleted != 4 {
		t.Fatalf("work result was not rebuilt: %#v", access)
	}
}

func TestReconstructSupportsEmptyAndBoundaryCursors(t *testing.T) {
	empty, err := Reconstruct(nil, 0, time.Minute)
	if err != nil || empty.Cursor != -1 || empty.EventCount != 0 {
		t.Fatalf("empty=%#v err=%v", empty, err)
	}
	base := time.Unix(3_000, 0).UTC()
	events := []protocol.Event{event("start", protocol.EventSessionStarted, "", base)}
	before, err := Reconstruct(events, -1, time.Minute)
	if err != nil || before.Cursor != -1 {
		t.Fatalf("before=%#v err=%v", before, err)
	}
	after, err := Reconstruct(events, 99, time.Minute)
	if err != nil || after.Cursor != 0 || after.EventCount != 1 {
		t.Fatalf("after=%#v err=%v", after, err)
	}
}

func event(id string, kind protocol.EventType, path string, at time.Time) protocol.Event {
	return protocol.Event{SchemaVersion: protocol.SchemaVersion, EventID: id, SessionID: "s", SourceType: protocol.SourceSynthetic, SourceConfidence: protocol.ConfidenceExact, EventType: kind, Timestamp: at, Path: path}
}
