package install

import (
	"encoding/json"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestMergeAndRemoveHooksPreservesUnrelatedEntries(t *testing.T) {
	original := []byte(`{
  "description": "keep me",
  "hooks": {
    "SessionStart": [{
      "matcher": "startup",
      "hooks": [{"type":"command","command":"existing","statusMessage":"Existing"}]
    }],
    "Notification": [{
      "hooks": [{"type":"command","command":"notify"}]
    }]
  }
}`)
	binary := filepath.Join(t.TempDir(), "path with spaces", "aav-codex-hook"+testExecutableSuffix())
	merged, removed, err := mergeHooks(original, binary)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 0 {
		t.Fatalf("removed = %d", removed)
	}
	if count, err := managedHookCount(merged); err != nil || count != len(hookEvents) {
		t.Fatalf("managed count = %d, err = %v", count, err)
	}
	if !strings.Contains(string(merged), `"description": "keep me"`) ||
		!strings.Contains(string(merged), `"command": "existing"`) ||
		!strings.Contains(string(merged), `"Notification"`) {
		t.Fatalf("unrelated configuration was lost:\n%s", merged)
	}
	cleaned, count, err := removeHooks(merged)
	if err != nil {
		t.Fatal(err)
	}
	if count != len(hookEvents) {
		t.Fatalf("removed = %d", count)
	}
	if !semanticEqual(cleaned, original) {
		t.Fatalf("cleaned configuration differs:\n%s", cleaned)
	}
}

func TestRemoveManagedHandlerPreservesHandlerAddedToManagedGroup(t *testing.T) {
	merged, _, err := mergeHooks(nil, filepath.Join(t.TempDir(), "hook"))
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(merged, &root); err != nil {
		t.Fatal(err)
	}
	var hooks map[string][]map[string]json.RawMessage
	if err := json.Unmarshal(root["hooks"], &hooks); err != nil {
		t.Fatal(err)
	}
	var handlers []json.RawMessage
	if err := json.Unmarshal(hooks["SessionStart"][0]["hooks"], &handlers); err != nil {
		t.Fatal(err)
	}
	handlers = append(handlers, json.RawMessage(`{"type":"command","command":"later","statusMessage":"Later"}`))
	hooks["SessionStart"][0]["hooks"], _ = json.Marshal(handlers)
	root["hooks"], _ = json.Marshal(hooks)
	modified, _ := json.Marshal(root)

	cleaned, removed, err := removeHooks(modified)
	if err != nil {
		t.Fatal(err)
	}
	if removed != len(hookEvents) || !strings.Contains(string(cleaned), `"later"`) {
		t.Fatalf("removed = %d, cleaned = %s", removed, cleaned)
	}
}

func TestManagedHandlerUsesPlatformQuoting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "quoted path", "hook"+testExecutableSuffix())
	handler, err := managedHandler(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(handler.Command, "'") || !strings.Contains(handler.Command, installID) {
		t.Fatalf("POSIX command = %q", handler.Command)
	}
	if !strings.HasPrefix(handler.CommandWindows, `"`) || !strings.Contains(handler.CommandWindows, `" --aav-install-id "`) {
		t.Fatalf("Windows command = %q", handler.CommandWindows)
	}
}

func TestInvalidConfigurationIsRejected(t *testing.T) {
	for _, payload := range [][]byte{
		[]byte(`[]`),
		[]byte(`{"hooks":[]}`),
		[]byte(`{"hooks":{}} trailing`),
	} {
		if _, _, err := mergeHooks(payload, "hook"); err == nil {
			t.Fatalf("expected error for %s", payload)
		}
	}
}

func testExecutableSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}
