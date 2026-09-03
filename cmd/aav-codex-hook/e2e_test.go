package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	workdiff "github.com/greadee/agent-action-visualizer/internal/diff"
	"github.com/greadee/agent-action-visualizer/internal/ingest"
	"github.com/greadee/agent-action-visualizer/internal/ipc"
	"github.com/greadee/agent-action-visualizer/internal/session"
	"github.com/greadee/agent-action-visualizer/internal/store"
	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

func TestReproducibleCodexFixtureSession(t *testing.T) {
	root := t.TempDir()
	initFixtureRepository(t, root)
	if err := os.WriteFile(filepath.Join(root, "main.txt"), []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "seed.txt"), []byte("fixture\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	commitFixture(t, root, "initial fixture")

	journalFile := filepath.Join(t.TempDir(), "session.db")
	journal, err := store.Open(journalFile)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = journal.Close() })
	recorder := newFixtureRecorder(t, root, journal)
	endpoint := hookTestEndpoint(t.Name())
	server := ipc.NewServer(endpoint, recorder.submit)
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	runFixtureHook(t, endpoint, fixtureInput("SessionStart", root, nil))
	waitForFixtureEvents(t, recorder, 1)
	runFixtureHook(t, endpoint, fixtureInput("PostToolUse", root, map[string]any{
		"turn_id": "turn-read", "tool_name": "read_file", "tool_use_id": "read-seed",
		"tool_input": map[string]any{"path": "seed.txt"},
	}))
	waitForFixtureEvents(t, recorder, 3)

	// A disconnected collector must silently lose only visualization evidence.
	server.Close()
	runFixtureHook(t, endpoint, fixtureInput("PostToolUse", root, map[string]any{
		"turn_id": "turn-disconnected", "tool_name": "read_file", "tool_use_id": "read-missing",
		"tool_input": map[string]any{"path": "main.txt"},
	}))
	if recorder.eventCount() != 3 { // session start, tool completion, file read
		t.Fatalf("disconnected hook changed collector state: %d events", recorder.eventCount())
	}

	server = ipc.NewServer(endpoint, recorder.submit)
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	if err := os.WriteFile(filepath.Join(root, "created.txt"), []byte("created\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runFixtureHook(t, endpoint, fixtureInput("PostToolUse", root, map[string]any{
		"turn_id": "turn-create", "tool_name": "create_file", "tool_use_id": "create-file",
		"tool_input": map[string]any{"path": "created.txt"},
	}))
	waitForFixtureEvents(t, recorder, 5)
	if err := os.WriteFile(filepath.Join(root, "main.txt"), []byte("new\nextra\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runFixtureHook(t, endpoint, fixtureInput("PostToolUse", root, map[string]any{
		"turn_id": "turn-patch", "tool_name": "apply_patch", "tool_use_id": "patch-main",
		"tool_input": map[string]any{"command": "*** Begin Patch\n*** Update File: main.txt\n@@\n-old\n+new\n+extra\n*** End Patch"},
	}))
	waitForFixtureEvents(t, recorder, 7)
	if err := os.Rename(filepath.Join(root, "created.txt"), filepath.Join(root, "moved.txt")); err != nil {
		t.Fatal(err)
	}
	runFixtureHook(t, endpoint, fixtureInput("PostToolUse", root, map[string]any{
		"turn_id": "turn-move", "tool_name": "move_file", "tool_use_id": "move-file",
		"tool_input": map[string]any{"source": "created.txt", "destination": "moved.txt"},
	}))
	waitForFixtureEvents(t, recorder, 9)
	if err := os.Remove(filepath.Join(root, "moved.txt")); err != nil {
		t.Fatal(err)
	}
	runFixtureHook(t, endpoint, fixtureInput("PostToolUse", root, map[string]any{
		"turn_id": "turn-delete", "tool_name": "delete_file", "tool_use_id": "delete-file",
		"tool_input": map[string]any{"path": "moved.txt"},
	}))
	waitForFixtureEvents(t, recorder, 11)
	runFixtureHook(t, endpoint, fixtureInput("SessionEnd", root, map[string]any{"reason": "other"}))
	waitForFixtureEvents(t, recorder, 12)

	state := recorder.state(t, "fixture-session")
	if state.StoppedAt == nil || len(state.Intervals) != 4 {
		t.Fatalf("session lifecycle state=%+v", state)
	}
	if state.Intervals[0].Path != "seed.txt" || state.Intervals[1].Path != "created.txt" ||
		state.Intervals[2].Path != "main.txt" || state.Intervals[3].Path != "moved.txt" {
		t.Fatalf("access trail=%+v", state.Intervals)
	}
	if state.PreviousPath != "main.txt" || state.ActivePath != "moved.txt" {
		t.Fatalf("focus state=%+v", state)
	}
	patch := state.Intervals[2]
	if patch.LinesAdded == nil || *patch.LinesAdded != 2 || patch.LinesDeleted == nil || *patch.LinesDeleted != 1 ||
		patch.WorkStatus != workdiff.StatusKnown || patch.WorkSource != workdiff.SourceStructuredPatch {
		t.Fatalf("structured work state=%+v", patch)
	}
	for _, interval := range state.Intervals {
		if interval.EndedAt == nil || interval.Duration < 0 {
			t.Fatalf("closed interval=%+v", interval)
		}
	}

	events, err := journal.Events(context.Background(), "fixture-session")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != recorder.eventCount() || containsSourceContent(events) {
		t.Fatalf("persisted events=%d recorder=%d sourceContent=%t", len(events), recorder.eventCount(), containsSourceContent(events))
	}
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}
	journal, err = store.Open(journalFile)
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	persisted, err := journal.Session(context.Background(), "fixture-session")
	if err != nil || persisted.StoppedAt == nil || len(persisted.Intervals) != len(state.Intervals) {
		t.Fatalf("persisted session=%+v err=%v", persisted, err)
	}
	replayed := session.NewEngine(2 * time.Minute)
	for _, event := range events {
		if _, err := replayed.Apply(event); err != nil {
			t.Fatalf("replay %s: %v", event.EventID, err)
		}
	}
	replayedState, err := replayed.State("fixture-session")
	if err != nil || replayedState.ActivePath != state.ActivePath || len(replayedState.Intervals) != len(state.Intervals) {
		t.Fatalf("replay state=%+v err=%v", replayedState, err)
	}
}

type fixtureRecorder struct {
	t       *testing.T
	root    string
	journal *store.Store
	engine  *session.Engine
	mu      sync.Mutex
	events  []protocol.Event
}

func newFixtureRecorder(t *testing.T, root string, journal *store.Store) *fixtureRecorder {
	return &fixtureRecorder{t: t, root: root, journal: journal, engine: session.NewEngine(2 * time.Minute)}
}

func (r *fixtureRecorder) submit(event protocol.Event) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	normalized, err := ingest.Normalize(event, r.root)
	if err != nil {
		r.t.Errorf("normalize %s: %v", event.EventID, err)
		return false
	}
	state, err := r.engine.Apply(normalized)
	if err != nil {
		r.t.Errorf("apply %s: %v", normalized.EventID, err)
		return false
	}
	if workFixtureEvent(normalized) && len(state.Intervals) > 0 {
		interval := state.Intervals[len(state.Intervals)-1]
		if interval.Path == normalized.Path && normalized.LinesAdded != nil && normalized.LinesDeleted != nil {
			state, err = r.engine.SetWorkResult(normalized.SessionID, interval.Sequence, normalized.LinesAdded, normalized.LinesDeleted, workdiff.StatusKnown, workdiff.SourceStructuredPatch, normalized.SourceConfidence)
			if err != nil {
				r.t.Errorf("set work result %s: %v", normalized.EventID, err)
				return false
			}
		}
	}
	if err := r.journal.SaveEvent(context.Background(), normalized); err != nil {
		r.t.Errorf("persist event %s: %v", normalized.EventID, err)
		return false
	}
	if err := r.journal.SaveSession(context.Background(), state, ""); err != nil {
		r.t.Errorf("persist session %s: %v", normalized.SessionID, err)
		return false
	}
	r.events = append(r.events, normalized)
	return true
}

func (r *fixtureRecorder) eventCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.events)
}

func (r *fixtureRecorder) state(t *testing.T, id string) session.State {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	state, err := r.engine.State(id)
	if err != nil {
		t.Fatal(err)
	}
	return state
}

func waitForFixtureEvents(t *testing.T, recorder *fixtureRecorder, expected int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if recorder.eventCount() >= expected {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d events; received %d", expected, recorder.eventCount())
}

func runFixtureHook(t *testing.T, endpoint string, input map[string]any) {
	t.Helper()
	payload, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(os.Args[0], "-test.run=TestHookHelperProcess")
	command.Env = append(os.Environ(), "AAV_HOOK_HELPER=1", "AAV_COLLECTOR_ENDPOINT="+endpoint)
	command.Stdin = strings.NewReader(string(payload))
	output, err := command.CombinedOutput()
	if err != nil || len(output) != 0 {
		t.Fatalf("hook err=%v output=%q", err, output)
	}
}

func fixtureInput(event, root string, values map[string]any) map[string]any {
	input := map[string]any{
		"session_id": "fixture-session", "cwd": root, "hook_event_name": event,
		"source": "startup", "permission_mode": "default",
	}
	for key, value := range values {
		input[key] = value
	}
	return input
}

func workFixtureEvent(event protocol.Event) bool {
	return event.EventType == protocol.EventFileCreated || event.EventType == protocol.EventFilePatched || event.EventType == protocol.EventFileModified || event.EventType == protocol.EventFileDeleted
}

func containsSourceContent(events []protocol.Event) bool {
	payload, _ := json.Marshal(events)
	return strings.Contains(string(payload), "old\\n") || strings.Contains(string(payload), "extra\\n")
}

func initFixtureRepository(t *testing.T, root string) {
	t.Helper()
	runGitFixture(t, root, "init")
	runGitFixture(t, root, "config", "user.email", "fixture@example.invalid")
	runGitFixture(t, root, "config", "user.name", "Fixture")
}

func commitFixture(t *testing.T, root, message string) {
	t.Helper()
	runGitFixture(t, root, "add", ".")
	runGitFixture(t, root, "commit", "-m", message)
}

func runGitFixture(t *testing.T, root string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v output=%s", strings.Join(args, " "), err, output)
	}
}
