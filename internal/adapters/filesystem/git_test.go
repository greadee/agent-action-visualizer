package filesystem

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestParseStatusAndNumstat(t *testing.T) {
	status := make(map[string]GitStatus)
	renames := make(map[string]string)
	parseStatus([]byte(" M src/a.go\x00R  src/new.go\x00src/old.go\x00?? src/new file.go\x00"), status, renames)
	if value := status["src/a.go"]; value.Index != ' ' || value.Worktree != 'M' {
		t.Fatalf("modified status = %#v", value)
	}
	if renames["src/old.go"] != "src/new.go" {
		t.Fatalf("renames = %#v", renames)
	}
	if value := status["src/new file.go"]; value.Index != '?' || value.Worktree != '?' {
		t.Fatalf("untracked status = %#v", value)
	}

	deltas := make(map[string]GitDelta)
	parseNumstat([]byte("4\t2\tsrc/a.go\x00-\t-\tasset.bin\x00"), deltas)
	if value := deltas["src/a.go"]; value.Added != 4 || value.Deleted != 2 || value.Binary {
		t.Fatalf("text delta = %#v", value)
	}
	if !deltas["asset.bin"].Binary {
		t.Fatalf("binary delta = %#v", deltas["asset.bin"])
	}
}

func TestRelativePathsDeduplicatesAndRejectsEscapes(t *testing.T) {
	root := t.TempDir()
	values := relativePaths(root, []string{
		root + "/src/a.go",
		root + "/src/a.go",
		root + "/../outside.go",
	})
	if len(values) != 1 || values[0] != "src/a.go" {
		t.Fatalf("relative paths = %#v", values)
	}
}

func TestCommandGitInspectorBatchesRepositoryEvidence(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	root := t.TempDir()
	runGit(t, root, "init", "--quiet")
	path := filepath.Join(root, "tracked.txt")
	if err := os.WriteFile(path, []byte("one\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "add", "tracked.txt")
	runGit(t, root, "-c", "user.name=AAV Test", "-c", "user.email=aav@example.invalid", "commit", "--quiet", "-m", "initial")
	if err := os.WriteFile(path, []byte("one\ntwo\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	inspector := NewCommandGitInspector(time.Second)
	evidence, err := inspector.Inspect(context.Background(), root, []string{path, path})
	if err != nil {
		t.Fatal(err)
	}
	if status := evidence.Status["tracked.txt"]; status.Worktree != 'M' {
		t.Fatalf("status = %#v", status)
	}
	if delta := evidence.Deltas["tracked.txt"]; delta.Added != 1 || delta.Deleted != 0 || delta.Binary {
		t.Fatalf("delta = %#v", delta)
	}
}

func runGit(t *testing.T, root string, arguments ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", root}, arguments...)...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", arguments, err, output)
	}
}
