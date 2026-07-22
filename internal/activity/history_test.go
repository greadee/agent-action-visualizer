package activity

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/greadee/agent-action-visualizer/internal/project"
)

func TestEnrichCombinesGitInferenceAndAgentReport(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.name", "AAV Test")
	runGit(t, root, "config", "user.email", "aav@example.invalid")
	write(t, filepath.Join(root, "app.go"), "package main\n")
	runGit(t, root, "add", "app.go")
	commit(t, root, "2026-01-01T12:00:00Z", "add app [tool:apply_patch]")
	write(t, filepath.Join(root, "app.go"), "package main\nfunc main() {}\n")
	runGit(t, root, "add", "app.go")
	commit(t, root, "2026-01-02T12:00:00Z", "fix app [tool:codex]")
	if err := os.MkdirAll(filepath.Join(root, ".aav"), 0o700); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(root, ".aav", "activity-v1.json"), `{"schema_version":1,"files":{"app.go":{"total_time_ms":90000,"lines_added":12,"lines_deleted":3,"recent_tools":["apply_patch"],"session_history":["session-review"]}}}`)

	nodes := []project.Node{
		{Path: ".", Kind: project.KindRoot},
		{Path: "app.go", ParentPath: ".", Kind: project.KindSource},
	}
	enriched, err := Enrich(context.Background(), root, nodes)
	if err != nil {
		t.Fatal(err)
	}
	activity := enriched[1].Activity
	if activity.AccessCount != 2 || activity.TotalTimeMS != 90000 || activity.LinesAdded != 12 {
		t.Fatalf("unexpected activity: %#v", activity)
	}
	if activity.GroupKey == "" || activity.LastCommitAt == 0 || activity.LastEvent != "commit "+shortCommit(activity.LastCommit)+": fix app [tool:codex]" {
		t.Fatalf("missing latest involvement: %#v", activity)
	}
	if len(activity.RecentTools) < 3 || activity.RecentTools[0] != "apply_patch" {
		t.Fatalf("tools were not merged: %#v", activity.RecentTools)
	}
	if enriched[0].Activity.AccessCount != 2 || enriched[0].Activity.TotalTimeMS != 90000 {
		t.Fatalf("root aggregate missing: %#v", enriched[0].Activity)
	}
}

func TestReadReportRejectsInvalidMetricsAndPaths(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".aav"), 0o700); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(root, ".aav", "activity-v1.json"), `{"schema_version":1,"files":{"../outside":{"total_time_ms":-1}}}`)
	if _, err := readReport(root); err == nil {
		t.Fatal("expected invalid agent report to fail")
	}
}

func runGit(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}

func commit(t *testing.T, root, date, message string) {
	t.Helper()
	cmd := exec.Command("git", "-C", root, "commit", "-m", message)
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_DATE="+date, "GIT_COMMITTER_DATE="+date)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, output)
	}
}

func write(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}
