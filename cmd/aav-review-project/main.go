// Command aav-review-project creates a deterministic Git repository for reviewing history clusters and activity extrusions.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

type fileActivity struct {
	TotalTimeMS    int64    `json:"total_time_ms"`
	LinesAdded     int64    `json:"lines_added"`
	LinesDeleted   int64    `json:"lines_deleted"`
	RecentTools    []string `json:"recent_tools"`
	SessionHistory []string `json:"session_history"`
}

func main() {
	out := flag.String("out", filepath.Join(os.TempDir(), "aav-extrusion-review"), "empty output directory")
	flag.Parse()
	if err := generate(*out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(*out)
}

func generate(root string) error {
	entries, err := os.ReadDir(root)
	if err == nil && len(entries) > 0 {
		return fmt.Errorf("output directory must be empty: %s", root)
	}
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	if err := git(root, "init"); err != nil {
		return err
	}
	if err := git(root, "config", "user.name", "AAV Review Agent"); err != nil {
		return err
	}
	if err := git(root, "config", "user.email", "review@aav.invalid"); err != nil {
		return err
	}

	if err := writeFiles(root, map[string]string{
		"src/core/parser.ts": "export const parse = (value: string) => value.trim()\n",
		"src/core/model.ts":  "export interface Item { id: string; value: string }\n",
		"app.config.json":    "{\"theme\":\"dark\"}\n",
		".gitignore":         ".aav/\n",
	}); err != nil {
		return err
	}
	if err := commit(root, "2026-06-01T10:00:00Z", "add core model [tool:apply_patch]"); err != nil {
		return err
	}

	if err := writeFiles(root, map[string]string{
		"src/ui/Dashboard.tsx": "export const Dashboard = () => <main>Review</main>\n",
		"src/ui/Graph.tsx":     "export const Graph = () => <canvas />\n",
		"src/ui/theme.css":     "body { background: #070a12; color: #edf4f7; }\n",
	}); err != nil {
		return err
	}
	if err := commit(root, "2026-06-04T14:00:00Z", "add graph screen [tool:codex]"); err != nil {
		return err
	}

	if err := writeFiles(root, map[string]string{
		"src/core/parser.ts":   "export const parse = (value: string) => value.trim().toLowerCase()\n",
		"src/core/model.ts":    "export interface Item { id: string; value: string; updatedAt: number }\n",
		"tests/parser.test.ts": "import { parse } from '../src/core/parser'\nvoid parse(' A ')\n",
	}); err != nil {
		return err
	}
	if err := commit(root, "2026-06-08T09:30:00Z", "fix parser behavior [tool:pytest]"); err != nil {
		return err
	}

	if err := writeFiles(root, map[string]string{
		"src/ui/Dashboard.tsx": "export const Dashboard = () => <main><h1>Activity</h1></main>\n",
		"src/ui/Graph.tsx":     "export const Graph = () => <canvas aria-label=\"project activity\" />\n",
	}); err != nil {
		return err
	}
	if err := commit(root, "2026-06-12T16:15:00Z", "upd activity graph [tool:apply_patch]"); err != nil {
		return err
	}

	report := struct {
		SchemaVersion int                     `json:"schema_version"`
		Files         map[string]fileActivity `json:"files"`
	}{1, map[string]fileActivity{
		"src/core/parser.ts":   {780000, 84, 22, []string{"apply_patch", "pytest"}, []string{"core-build", "parser-fix"}},
		"src/core/model.ts":    {420000, 51, 9, []string{"apply_patch"}, []string{"core-build"}},
		"tests/parser.test.ts": {240000, 38, 4, []string{"pytest"}, []string{"parser-fix"}},
		"src/ui/Dashboard.tsx": {960000, 132, 31, []string{"apply_patch", "codex"}, []string{"ui-build", "activity-polish"}},
		"src/ui/Graph.tsx":     {1140000, 176, 45, []string{"apply_patch", "codex"}, []string{"ui-build", "activity-polish"}},
		"src/ui/theme.css":     {180000, 29, 7, []string{"codex"}, []string{"ui-build"}},
		"app.config.json":      {60000, 8, 2, []string{"apply_patch"}, []string{"core-build"}},
	}}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	base := time.Date(2026, 6, 12, 16, 20, 0, 0, time.UTC)
	focusEvents := []protocol.Event{
		reviewEvent("focus-start", protocol.EventSessionStarted, "", base, nil),
		reviewEvent("focus-dashboard", protocol.EventFilePatched, "src/ui/Dashboard.tsx", base.Add(time.Second), []string{"src/ui/Graph.tsx", "src/ui/theme.css"}),
		reviewEvent("focus-graph", protocol.EventFileRead, "src/ui/Graph.tsx", base.Add(2*time.Second), []string{"src/ui/Dashboard.tsx", "tests/parser.test.ts"}),
		reviewEvent("focus-parser", protocol.EventFileModified, "src/core/parser.ts", base.Add(3*time.Second), []string{"src/core/model.ts", "tests/parser.test.ts"}),
	}
	focusData, err := json.MarshalIndent(focusEvents, "", "  ")
	if err != nil {
		return err
	}
	denseEvents := accessPointReviewEvents(base)
	denseData, err := json.MarshalIndent(denseEvents, "", "  ")
	if err != nil {
		return err
	}
	return writeFiles(root, map[string]string{
		".aav/activity-v1.json":             string(data) + "\n",
		".aav/focus-review-v1.json":         string(focusData) + "\n",
		".aav/access-points-review-v1.json": string(denseData) + "\n",
	})
}

func accessPointReviewEvents(base time.Time) []protocol.Event {
	events := []protocol.Event{{
		SchemaVersion:    protocol.SchemaVersion,
		EventID:          "access-points-start",
		SessionID:        "review-access-points",
		SourceType:       protocol.SourceSynthetic,
		SourceConfidence: protocol.ConfidenceExact,
		EventType:        protocol.EventSessionStarted,
		Timestamp:        base,
	}}
	paths := []string{"src/ui/Graph.tsx", "src/ui/Dashboard.tsx"}
	for index := 0; index < 60; index++ {
		eventType := protocol.EventFileRead
		operation := "read"
		if index%2 == 0 {
			eventType, operation = protocol.EventFilePatched, "patch"
		}
		events = append(events, protocol.Event{
			SchemaVersion:    protocol.SchemaVersion,
			EventID:          fmt.Sprintf("access-point-%02d", index+1),
			SessionID:        "review-access-points",
			SourceType:       protocol.SourceSynthetic,
			SourceConfidence: protocol.ConfidenceExact,
			EventType:        eventType,
			Operation:        operation,
			Timestamp:        base.Add(time.Duration(index+1) * time.Second),
			Path:             paths[index%len(paths)],
		})
	}
	return events
}

func reviewEvent(id string, eventType protocol.EventType, path string, at time.Time, secondary []string) protocol.Event {
	metadata := map[string]interface{}(nil)
	if len(secondary) > 0 {
		metadata = map[string]interface{}{"secondary_paths": secondary}
	}
	return protocol.Event{
		SchemaVersion:    protocol.SchemaVersion,
		EventID:          id,
		SessionID:        "review-live-focus",
		SourceType:       protocol.SourceSynthetic,
		SourceConfidence: protocol.ConfidenceExact,
		EventType:        eventType,
		Operation:        string(eventType),
		Timestamp:        at,
		Path:             path,
		Metadata:         metadata,
	}
}

func writeFiles(root string, files map[string]string) error {
	for path, contents := range files {
		absolute := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(absolute), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(absolute, []byte(contents), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func commit(root, date, message string) error {
	if err := git(root, "add", "."); err != nil {
		return err
	}
	cmd := exec.Command("git", "-C", root, "commit", "-m", message)
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_DATE="+date, "GIT_COMMITTER_DATE="+date)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git commit: %w: %s", err, output)
	}
	return nil
}

func git(root string, args ...string) error {
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %v: %w: %s", args, err, output)
	}
	return nil
}
