package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	workdiff "github.com/greadee/agent-action-visualizer/internal/diff"
	localipc "github.com/greadee/agent-action-visualizer/internal/ipc"
	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

func TestLoadAndRefreshProject(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main"), 0o600); err != nil {
		t.Fatal(err)
	}
	app := newTestApp(t)
	initial, err := app.LoadProject(root)
	if err != nil {
		t.Fatal(err)
	}
	if initial.Revision != 1 || len(initial.Nodes) != 2 {
		t.Fatalf("unexpected initial snapshot: %#v", initial)
	}
	if err := os.WriteFile(filepath.Join(root, "main_test.go"), []byte("package main"), 0o600); err != nil {
		t.Fatal(err)
	}
	patch, err := app.RefreshProject()
	if err != nil {
		t.Fatal(err)
	}
	if patch.Revision != 2 || len(patch.Added) != 1 {
		t.Fatalf("unexpected refresh patch: %#v", patch)
	}
}

func TestPublishActivityEventMapsLiveFocusNodes(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"main.go", "helper.go", "README.md"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	app := newTestApp(t)
	graph, err := app.LoadProject(root)
	if err != nil {
		t.Fatal(err)
	}
	ids := make(map[string]string)
	for _, node := range graph.Nodes {
		ids[node.Path] = node.ID
	}
	base := time.Unix(1000, 0).UTC()
	start := activityEvent("start", protocol.EventSessionStarted, "", base)
	if _, err := app.PublishActivityEvent(start); err != nil {
		t.Fatal(err)
	}
	first := activityEvent("read", protocol.EventFileRead, "main.go", base.Add(time.Second))
	if _, err := app.PublishActivityEvent(first); err != nil {
		t.Fatal(err)
	}
	second := activityEvent("patch", protocol.EventFilePatched, "helper.go", base.Add(2*time.Second))
	second.Metadata = map[string]interface{}{"secondary_paths": []interface{}{"README.md", "main.go"}}
	focus, err := app.PublishActivityEvent(second)
	if err != nil {
		t.Fatal(err)
	}
	if focus.ActiveNodeID != ids["helper.go"] || focus.PreviousNodeID != ids["main.go"] {
		t.Fatalf("incorrect primary focus mapping: %#v", focus)
	}
	if focus.Operation != string(protocol.EventFilePatched) || focus.Confidence != protocol.ConfidenceExact {
		t.Fatalf("incorrect operation details: %#v", focus)
	}
	secondary := map[string]bool{}
	for _, id := range focus.SecondaryNodeIDs {
		secondary[id] = true
	}
	if len(secondary) != 2 || !secondary[ids["main.go"]] || !secondary[ids["README.md"]] {
		t.Fatalf("incorrect secondary mapping: %#v", focus)
	}
	if len(focus.Trail) != 2 {
		t.Fatalf("incorrect trail length: %#v", focus.Trail)
	}
	if focus.Trail[0].Sequence != 1 || focus.Trail[0].NodeID != ids["main.go"] || focus.Trail[0].EndedAt == nil || focus.Trail[0].DurationMS != 1000 {
		t.Fatalf("incorrect closed trail access: %#v", focus.Trail[0])
	}
	if focus.Trail[1].Sequence != 2 || focus.Trail[1].NodeID != ids["helper.go"] || focus.Trail[1].EndedAt != nil || len(focus.Trail[1].Operations) != 1 {
		t.Fatalf("incorrect active trail access: %#v", focus.Trail[1])
	}
}

func TestPublishActivityEventRejectsEscapingSecondaryPath(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main"), 0o600); err != nil {
		t.Fatal(err)
	}
	app := newTestApp(t)
	if _, err := app.LoadProject(root); err != nil {
		t.Fatal(err)
	}
	base := time.Unix(1000, 0).UTC()
	if _, err := app.PublishActivityEvent(activityEvent("start", protocol.EventSessionStarted, "", base)); err != nil {
		t.Fatal(err)
	}
	event := activityEvent("read", protocol.EventFileRead, "main.go", base.Add(time.Second))
	event.Metadata = map[string]interface{}{"secondary_paths": []string{"../secret.txt"}}
	if _, err := app.PublishActivityEvent(event); err == nil {
		t.Fatal("expected escaping secondary path to be rejected")
	}
}

func TestStructuredWorkDeltaUpdatesAsynchronously(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main"), 0o600); err != nil {
		t.Fatal(err)
	}
	app := newTestApp(t)
	if _, err := app.LoadProject(root); err != nil {
		t.Fatal(err)
	}
	base := time.Unix(1000, 0).UTC()
	if _, err := app.PublishActivityEvent(activityEvent("start", protocol.EventSessionStarted, "", base)); err != nil {
		t.Fatal(err)
	}
	added, deleted := int64(14), int64(6)
	event := activityEvent("work", protocol.EventFilePatched, "main.go", base.Add(time.Second))
	event.LinesAdded, event.LinesDeleted = &added, &deleted
	focus, err := app.PublishActivityEvent(event)
	if err != nil {
		t.Fatal(err)
	}
	if focus.Trail[0].WorkStatus != workdiff.StatusPending {
		t.Fatalf("submission blocked on delta work: %#v", focus.Trail[0])
	}
	deadline := time.Now().Add(time.Second)
	for {
		state, stateErr := app.focus.State("review-session")
		if stateErr != nil {
			t.Fatal(stateErr)
		}
		interval := state.Intervals[0]
		if interval.WorkStatus == workdiff.StatusKnown {
			if interval.LinesAdded == nil || *interval.LinesAdded != added ||
				interval.LinesDeleted == nil || *interval.LinesDeleted != deleted ||
				interval.WorkSource != workdiff.SourceStructuredPatch {
				t.Fatalf("incorrect asynchronous work result: %+v", interval)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for work result: %+v", interval)
		}
		time.Sleep(time.Millisecond)
	}
}

func TestLocalCollectorFeedsSessionPipeline(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main"), 0o600); err != nil {
		t.Fatal(err)
	}
	app := newTestApp(t)
	if _, err := app.LoadProject(root); err != nil {
		t.Fatal(err)
	}
	app.ipcEndpoint = desktopTestEndpoint(t.Name())
	app.startCollector()
	if app.Health()["collector"] != "ready" {
		t.Fatalf("collector health = %#v", app.Health())
	}
	base := time.Unix(5_000, 0).UTC()
	events := []protocol.Event{
		activityEvent("ipc-start", protocol.EventSessionStarted, "", base),
		activityEvent("ipc-read", protocol.EventFileRead, "main.go", base.Add(time.Second)),
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := localipc.NewClient(app.ipcEndpoint).Send(ctx, events); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for {
		state, err := app.focus.State("review-session")
		if err == nil && state.ActivePath == "main.go" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("collector event did not reach session pipeline: state=%#v err=%v", state, err)
		}
		time.Sleep(time.Millisecond)
	}
}

func TestNativeMoveAndDeletePreserveGraphIdentity(t *testing.T) {
	root := t.TempDir()
	oldPath := filepath.Join(root, "old.txt")
	if err := os.WriteFile(oldPath, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	app := newTestApp(t)
	initial, err := app.LoadProject(root)
	if err != nil {
		t.Fatal(err)
	}
	oldID := nodeIDForTest(initial, "old.txt")
	if oldID == "" {
		t.Fatalf("missing original node: %#v", initial)
	}
	base := time.Unix(6_000, 0).UTC()
	if _, err := app.PublishActivityEvent(activityEvent("start", protocol.EventSessionStarted, "", base)); err != nil {
		t.Fatal(err)
	}
	newPath := filepath.Join(root, "renamed.txt")
	if err := os.Rename(oldPath, newPath); err != nil {
		t.Fatal(err)
	}
	moved := activityEvent("move", protocol.EventFileMoved, "renamed.txt", base.Add(time.Second))
	moved.PreviousPath = "old.txt"
	focus, err := app.PublishActivityEvent(moved)
	if err != nil {
		t.Fatal(err)
	}
	if focus.ActiveNodeID != oldID || focus.ActivePath != "renamed.txt" {
		t.Fatalf("move focus=%#v", focus)
	}
	patch, err := app.RefreshProject()
	if err != nil {
		t.Fatal(err)
	}
	if nodeIDForTest(graphSnapshotDTO{Nodes: patch.Updated}, "renamed.txt") != oldID {
		t.Fatalf("rename patch=%#v", patch)
	}
	if err := os.Remove(newPath); err != nil {
		t.Fatal(err)
	}
	deleted := activityEvent("delete", protocol.EventFileDeleted, "renamed.txt", base.Add(2*time.Second))
	focus, err = app.PublishActivityEvent(deleted)
	if err != nil {
		t.Fatal(err)
	}
	if focus.ActiveNodeID != oldID || focus.ActivePath != "renamed.txt" {
		t.Fatalf("delete focus=%#v", focus)
	}
	patch, err = app.RefreshProject()
	if err != nil {
		t.Fatal(err)
	}
	if len(patch.Updated) != 1 || patch.Updated[0].ID != oldID || patch.Updated[0].Kind != "tombstone" {
		t.Fatalf("tombstone patch=%#v", patch)
	}
}

func newTestApp(t *testing.T) *App {
	t.Helper()
	app := NewApp()
	t.Cleanup(func() { app.shutdown(context.Background()) })
	return app
}

func activityEvent(id string, kind protocol.EventType, path string, at time.Time) protocol.Event {
	return protocol.Event{
		SchemaVersion:    protocol.SchemaVersion,
		EventID:          id,
		SessionID:        "review-session",
		SourceType:       protocol.SourceSynthetic,
		SourceConfidence: protocol.ConfidenceExact,
		EventType:        kind,
		Operation:        string(kind),
		Timestamp:        at,
		Path:             path,
	}
}

func desktopTestEndpoint(seed string) string {
	sum := sha256.Sum256([]byte(seed))
	if runtime.GOOS == "windows" {
		return `\\.\pipe\aav-desktop-test-` + hex.EncodeToString(sum[:8])
	}
	return filepath.Join(os.TempDir(), "aav-desktop-test-"+hex.EncodeToString(sum[:8])+".sock")
}

func nodeIDForTest(snapshot graphSnapshotDTO, path string) string {
	for _, node := range snapshot.Nodes {
		if node.Path == path {
			return node.ID
		}
	}
	return ""
}
