package codex

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

func TestTranslateApplyPatchIsMetadataOnly(t *testing.T) {
	input := fixtureHook(t, "hooks-apply-patch-v1.json")
	at := time.Unix(1_000, 0).UTC()
	events := Translate(input, at)
	if len(events) != 4 {
		t.Fatalf("events = %d, want tool plus three file events: %#v", len(events), events)
	}
	tool := events[0]
	if tool.EventType != protocol.EventToolCompleted || tool.SourceConfidence != protocol.ConfidenceExact || tool.CorrelationID != "tool-sanitized" {
		t.Fatalf("unexpected tool event: %#v", tool)
	}
	want := []struct {
		kind    protocol.EventType
		path    string
		added   int64
		deleted int64
	}{
		{protocol.EventFilePatched, "src/changed.go", 1, 1},
		{protocol.EventFileCreated, "src/created.go", 1, 0},
		{protocol.EventFileDeleted, "src/deleted.go", 0, 1},
	}
	for index, expected := range want {
		event := events[index+1]
		if event.EventType != expected.kind || filepath.ToSlash(event.Path) != "/workspace/fixture/"+expected.path {
			t.Fatalf("event %d identity = %#v", index, event)
		}
		if event.SourceConfidence != protocol.ConfidenceCorrelated || event.ParentEventID != tool.EventID {
			t.Fatalf("event %d provenance = %#v", index, event)
		}
		if event.LinesAdded == nil || *event.LinesAdded != expected.added || event.LinesDeleted == nil || *event.LinesDeleted != expected.deleted {
			t.Fatalf("event %d delta = %#v", index, event)
		}
	}
	payload, err := json.Marshal(events)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"old-sensitive-line", "new-sensitive-line", "tool_response", "model-sanitized", "transcript"} {
		if strings.Contains(string(payload), forbidden) {
			t.Fatalf("normalized events retained forbidden content %q: %s", forbidden, payload)
		}
	}
}

func TestTranslateLifecycleAndCorrelation(t *testing.T) {
	at := time.Unix(2_000, 0).UTC()
	start := Translate(HookInput{SessionID: "session", HookEventName: "SessionStart", Source: "startup"}, at)
	if len(start) != 1 || start[0].EventType != protocol.EventSessionStarted || start[0].SourceConfidence != protocol.ConfidenceExact {
		t.Fatalf("session start = %#v", start)
	}
	if compact := Translate(HookInput{SessionID: "session", HookEventName: "SessionStart", Source: "compact"}, at); len(compact) != 0 {
		t.Fatalf("compact must not reset session state: %#v", compact)
	}
	pre := Translate(HookInput{SessionID: "session", TurnID: "turn", HookEventName: "PreToolUse", ToolName: "Bash", ToolUseID: "tool"}, at)
	post := Translate(HookInput{SessionID: "session", TurnID: "turn", HookEventName: "PostToolUse", ToolName: "Bash", ToolUseID: "tool"}, at.Add(time.Second))
	if len(pre) != 1 || len(post) != 1 || pre[0].EventType != protocol.EventToolStarted || post[0].EventType != protocol.EventToolCompleted {
		t.Fatalf("tool lifecycle pre=%#v post=%#v", pre, post)
	}
	if pre[0].CorrelationID != post[0].CorrelationID || pre[0].EventID == post[0].EventID {
		t.Fatalf("correlation pre=%#v post=%#v", pre[0], post[0])
	}
}

func TestTranslateRecognizedAndUnknownStructuredTools(t *testing.T) {
	at := time.Unix(3_000, 0).UTC()
	read := Translate(HookInput{
		SessionID: "session", HookEventName: "PostToolUse", ToolName: "mcp__filesystem__read_file", ToolUseID: "read",
		CWD: "/workspace", ToolInput: json.RawMessage(`{"path":"src/a.go","content":"do-not-retain"}`), ToolResponse: json.RawMessage(`"secret response"`),
	}, at)
	if len(read) != 2 || read[1].EventType != protocol.EventFileRead || filepath.ToSlash(read[1].Path) != "/workspace/src/a.go" {
		t.Fatalf("read events = %#v", read)
	}
	unknown := Translate(HookInput{
		SessionID: "session", HookEventName: "PostToolUse", ToolName: "mcp__custom__analyze", ToolUseID: "unknown",
		ToolInput: json.RawMessage(`{"path":"src/not-evidence.go"}`),
	}, at)
	if len(unknown) != 1 || unknown[0].EventType != protocol.EventToolCompleted {
		t.Fatalf("unknown tool invented file evidence: %#v", unknown)
	}
}

func TestTranslateRenameAndPathBounds(t *testing.T) {
	at := time.Unix(4_000, 0).UTC()
	events := Translate(HookInput{
		SessionID: "session", HookEventName: "PostToolUse", ToolName: "mcp__filesystem__move_file", ToolUseID: "move", CWD: "/workspace",
		ToolInput: json.RawMessage(`{"source":"src/old.go","destination":"lib/new.go"}`),
	}, at)
	if len(events) != 2 || events[1].EventType != protocol.EventFileMoved || filepath.ToSlash(events[1].PreviousPath) != "/workspace/src/old.go" || filepath.ToSlash(events[1].Path) != "/workspace/lib/new.go" {
		t.Fatalf("move events = %#v", events)
	}
	oversized := strings.Repeat("a", 4097)
	raw, _ := json.Marshal(map[string]string{"path": oversized})
	bounded := Translate(HookInput{SessionID: "session", HookEventName: "PostToolUse", ToolName: "read_file", ToolUseID: "bounded", ToolInput: raw}, at)
	if len(bounded) != 1 {
		t.Fatalf("oversized path should be omitted: %#v", bounded)
	}
}

func fixtureHook(t *testing.T, name string) HookInput {
	t.Helper()
	payload, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "codex", name))
	if err != nil {
		t.Fatal(err)
	}
	var input HookInput
	if err := json.Unmarshal(payload, &input); err != nil {
		t.Fatal(err)
	}
	return input
}
