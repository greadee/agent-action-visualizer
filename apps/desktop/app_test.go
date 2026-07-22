package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

func TestLoadAndRefreshProject(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main"), 0o600); err != nil {
		t.Fatal(err)
	}
	app := NewApp()
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
	app := NewApp()
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
}

func TestPublishActivityEventRejectsEscapingSecondaryPath(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main"), 0o600); err != nil {
		t.Fatal(err)
	}
	app := NewApp()
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
