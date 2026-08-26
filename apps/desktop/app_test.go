package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	workdiff "github.com/greadee/agent-action-visualizer/internal/diff"
	localipc "github.com/greadee/agent-action-visualizer/internal/ipc"
	"github.com/greadee/agent-action-visualizer/internal/session"
	"github.com/greadee/agent-action-visualizer/internal/store"
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

func TestOpenProjectValidatesUserSelectedRepository(t *testing.T) {
	repository := t.TempDir()
	if err := os.Mkdir(filepath.Join(repository, ".git"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repository, "README.md"), []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	filePath := filepath.Join(t.TempDir(), "not-a-folder.txt")
	if err := os.WriteFile(filePath, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	nonRepository := t.TempDir()
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "empty", path: "", want: "choose a local Git repository"},
		{name: "missing", path: filepath.Join(t.TempDir(), "missing"), want: "was not found"},
		{name: "file", path: filePath, want: "is not a folder"},
		{name: "non repository", path: nonRepository, want: "is not a Git repository"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app := newTestApp(t)
			if _, err := app.OpenProject(test.path); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("OpenProject(%q) error = %v, want %q", test.path, err, test.want)
			}
		})
	}
	app := newTestApp(t)
	opened, err := app.OpenProject(repository)
	if err != nil {
		t.Fatal(err)
	}
	if opened.Root != filepath.Clean(repository) || len(opened.Graph.Nodes) != 2 {
		t.Fatalf("opened repository = %#v", opened)
	}
}

func TestDisconnectedCollectorStillLoadsStaticProject(t *testing.T) {
	endpoint := desktopTestEndpoint(t.Name())
	owner := newTestApp(t)
	owner.ipcEndpoint = endpoint
	owner.startCollector()
	if owner.Health()["collector"] != "ready" {
		t.Fatalf("owner health = %#v", owner.Health())
	}

	repository := t.TempDir()
	if err := os.Mkdir(filepath.Join(repository, ".git"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repository, "main.go"), []byte("package main"), 0o600); err != nil {
		t.Fatal(err)
	}
	disconnected := newTestApp(t)
	disconnected.ipcEndpoint = endpoint
	disconnected.startCollector()
	if disconnected.Health()["collector"] != "unavailable" {
		t.Fatalf("disconnected health = %#v", disconnected.Health())
	}
	opened, err := disconnected.OpenProject(repository)
	if err != nil || len(opened.Graph.Nodes) != 2 {
		t.Fatalf("static project load = %#v, error = %v", opened, err)
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

func TestPersistedSessionReplayIsCursorBounded(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main"), 0o600); err != nil {
		t.Fatal(err)
	}
	app := newTestApp(t)
	if _, err := app.LoadProject(root); err != nil {
		t.Fatal(err)
	}
	app.startJournalAt(filepath.Join(t.TempDir(), "sessions.db"))
	base := time.Unix(7_000, 0).UTC()
	for _, event := range []protocol.Event{
		activityEvent("replay-start", protocol.EventSessionStarted, "", base),
		activityEvent("replay-read", protocol.EventFileRead, "main.go", base.Add(time.Second)),
	} {
		if _, err := app.PublishActivityEvent(event); err != nil {
			t.Fatal(err)
		}
	}
	deadline := time.Now().Add(time.Second)
	for {
		sessions, err := app.ListPersistedSessions()
		if err == nil && len(sessions) == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("persisted sessions did not arrive: sessions=%#v err=%v", sessions, err)
		}
		time.Sleep(time.Millisecond)
	}
	replay, err := app.ReplayPersistedSession("review-session", 99)
	if err != nil {
		t.Fatal(err)
	}
	if replay.Cursor != 1 || replay.EventCount != 2 || len(replay.Focus.Trail) != 1 || replay.Focus.Trail[0].EndedAt == nil || replay.Focus.Trail[0].DurationMS != 0 {
		t.Fatalf("unexpected replay=%#v", replay)
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

func TestResyncReturnsAuthoritativeGraphAndFocus(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main"), 0o600); err != nil {
		t.Fatal(err)
	}
	app := newTestApp(t)
	graph, err := app.LoadProject(root)
	if err != nil {
		t.Fatal(err)
	}
	base := time.Unix(8_000, 0).UTC()
	if _, err := app.PublishActivityEvent(activityEvent("resync-start", protocol.EventSessionStarted, "", base)); err != nil {
		t.Fatal(err)
	}
	if _, err := app.PublishActivityEvent(activityEvent("resync-read", protocol.EventFileRead, "main.go", base.Add(time.Second))); err != nil {
		t.Fatal(err)
	}
	state := app.Resync()
	if state.Graph.Revision != graph.Revision || state.Focus == nil || state.Focus.ActivePath != "main.go" || len(state.Focus.Trail) != 1 {
		t.Fatalf("resync state = %#v", state)
	}
}

func TestShutdownIsIdempotent(t *testing.T) {
	app := NewApp()
	app.shutdown(context.Background())
	app.shutdown(context.Background())
	if app.Health()["service"] != "agent-action-visualizer" {
		t.Fatalf("health after shutdown = %#v", app.Health())
	}
}

func TestJournalCorruptionRecoveryUsesSafeHealthCode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.db")
	app := newTestApp(t)
	const privateMarker = "private-database-payload"
	if err := os.WriteFile(path, []byte(privateMarker), 0o600); err != nil {
		t.Fatal(err)
	}
	app.startJournalAt(path)
	health := app.Health()
	if health["persistence"] != "ready" || health["recovery"] != "database_quarantined" {
		t.Fatalf("health = %#v", health)
	}
	for key, value := range health {
		if strings.Contains(value, privateMarker) || strings.Contains(value, path) {
			t.Fatalf("health field %s leaked recovery data", key)
		}
	}
}

func TestJournalStartupRecoversStaleSession(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.db")
	journal, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now().UTC().Add(-time.Hour)
	state := session.State{
		SessionID: "stale", StartedAt: started, ActivePath: "main.go",
		Intervals: []session.AccessInterval{{Sequence: 1, Path: "main.go", StartedAt: started}},
	}
	if err := journal.SaveSession(context.Background(), state, ""); err != nil {
		t.Fatal(err)
	}
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}
	app := newTestApp(t)
	app.startJournalAt(path)
	if app.Health()["recovery"] != "stale_sessions_closed" {
		t.Fatalf("health = %#v", app.Health())
	}
	app.stopJournal()
	journal, err = store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	recovered, err := journal.Session(context.Background(), "stale")
	if err != nil || recovered.StoppedAt == nil || recovered.Intervals[0].Duration != session.DefaultIdleTimeout {
		t.Fatalf("recovered state = %+v err=%v", recovered, err)
	}
}

func TestShutdownFlushesQueuedJournalRecords(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main"), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "sessions.db")
	app := NewApp()
	if _, err := app.LoadProject(root); err != nil {
		t.Fatal(err)
	}
	app.startJournalAt(path)
	base := time.Unix(9_000, 0).UTC()
	if _, err := app.PublishActivityEvent(activityEvent("flush-start", protocol.EventSessionStarted, "", base)); err != nil {
		t.Fatal(err)
	}
	if _, err := app.PublishActivityEvent(activityEvent("flush-read", protocol.EventFileRead, "main.go", base.Add(time.Second))); err != nil {
		t.Fatal(err)
	}
	app.shutdown(context.Background())
	journal, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	events, err := journal.Events(context.Background(), "review-session")
	if err != nil || len(events) != 2 {
		t.Fatalf("flushed events = %#v err=%v", events, err)
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
