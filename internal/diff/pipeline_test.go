package diff

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

func number(value int64) *int64 { return &value }

func request(key, path string) Request {
	return Request{
		Key: key, SessionID: "session", Sequence: 3, ProjectRoot: "repo",
		Path: path, Confidence: protocol.ConfidenceExact,
	}
}

func TestStructuredCountsTakePriority(t *testing.T) {
	value := "@@ -1 +1 @@\n-old\n+new\n"
	item := request("structured", "a.go")
	item.StructuredAdded = number(17)
	item.StructuredDeleted = number(4)
	item.StructuredPatch = &value
	item.AllowSnapshotCorrelation = true
	item.Before, item.After = []byte("old"), []byte("new")
	result, resolved := resolveDirect(item)
	if !resolved || result.Status != StatusKnown || *result.LinesAdded != 17 || *result.LinesDeleted != 4 {
		t.Fatalf("structured result = %+v", result)
	}
	if result.Source != SourceStructuredPatch || result.Confidence != protocol.ConfidenceExact {
		t.Fatalf("structured provenance = %+v", result)
	}
}

func TestPatchAndSnapshotStates(t *testing.T) {
	patch := "--- a.go\n+++ a.go\n@@ -1,2 +1,3 @@\n-old\n+new\n+extra\n keep\n"
	item := request("patch", "a.go")
	item.StructuredPatch = &patch
	result, resolved := resolveDirect(item)
	if !resolved || *result.LinesAdded != 2 || *result.LinesDeleted != 1 {
		t.Fatalf("patch result = %+v", result)
	}

	snapshot := request("snapshot", "a.go")
	snapshot.AllowSnapshotCorrelation = true
	snapshot.Before = []byte("one\ntwo\n")
	snapshot.After = []byte("one\nthree\nfour\n")
	result, resolved = resolveDirect(snapshot)
	if !resolved || *result.LinesAdded != 2 || *result.LinesDeleted != 1 || result.Source != SourceCorrelatedSnapshot {
		t.Fatalf("snapshot result = %+v", result)
	}
}

func TestBinaryUnsupportedEmptyAndExtremeStates(t *testing.T) {
	binary := request("binary", "image.png")
	binary.IsBinary = true
	result, _ := resolveDirect(binary)
	if result.Status != StatusBinary || result.LinesAdded != nil {
		t.Fatalf("binary result = %+v", result)
	}

	invalid := string([]byte{0xff, 0xfe})
	unsupported := request("encoding", "legacy.txt")
	unsupported.StructuredPatch = &invalid
	result, _ = resolveDirect(unsupported)
	if result.Status != StatusUnsupportedEncoding {
		t.Fatalf("encoding result = %+v", result)
	}

	empty := request("empty", "empty.go")
	empty.StructuredAdded, empty.StructuredDeleted = number(0), number(0)
	result, _ = resolveDirect(empty)
	if result.Status != StatusEmpty {
		t.Fatalf("empty result = %+v", result)
	}

	extreme := request("extreme", "large.go")
	extreme.StructuredAdded, extreme.StructuredDeleted = number(1_000_000), number(750_000)
	result, _ = resolveDirect(extreme)
	if *result.LinesAdded != 1_000_000 || *result.LinesDeleted != 750_000 {
		t.Fatalf("extreme result = %+v", result)
	}
}

type fakeGitRunner struct {
	mu      sync.Mutex
	calls   [][]string
	results map[string]GitDelta
	err     error
	started chan struct{}
	block   bool
}

func (f *fakeGitRunner) Numstat(ctx context.Context, _ string, paths []string) (map[string]GitDelta, error) {
	f.mu.Lock()
	f.calls = append(f.calls, append([]string(nil), paths...))
	f.mu.Unlock()
	if f.started != nil {
		select {
		case f.started <- struct{}{}:
		default:
		}
	}
	if f.block {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return f.results, f.err
}

func receive(t *testing.T, pipeline *Pipeline) Result {
	t.Helper()
	select {
	case result := <-pipeline.Results():
		return result
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for diff result")
		return Result{}
	}
}

func TestPipelineBatchesGitFallback(t *testing.T) {
	runner := &fakeGitRunner{results: map[string]GitDelta{
		"a.go": {Added: 4, Deleted: 1},
		"b.go": {Binary: true},
	}}
	pipeline := NewPipeline(8, 8, runner)
	defer pipeline.Close()
	if !pipeline.Submit(request("a", "a.go")) || !pipeline.Submit(request("b", "b.go")) {
		t.Fatal("requests were not accepted")
	}
	results := map[string]Result{}
	for range 2 {
		result := receive(t, pipeline)
		results[result.Path] = result
	}
	if results["a.go"].Source != SourceGitNumstat || *results["a.go"].LinesAdded != 4 {
		t.Fatalf("git result = %+v", results["a.go"])
	}
	if results["b.go"].Status != StatusBinary {
		t.Fatalf("binary git result = %+v", results["b.go"])
	}
	runner.mu.Lock()
	defer runner.mu.Unlock()
	if len(runner.calls) != 1 || len(runner.calls[0]) != 2 {
		t.Fatalf("git calls = %#v", runner.calls)
	}
}

func TestPipelineDeduplicatesBoundsAndCancels(t *testing.T) {
	runner := &fakeGitRunner{started: make(chan struct{}, 1), block: true}
	pipeline := NewPipeline(1, 1, runner)
	if !pipeline.Submit(request("first", "a.go")) {
		t.Fatal("first request rejected")
	}
	select {
	case <-runner.started:
	case <-time.After(time.Second):
		t.Fatal("runner did not start")
	}
	if !pipeline.Submit(request("queued", "b.go")) {
		t.Fatal("queued request rejected")
	}
	if pipeline.Submit(request("queued", "b.go")) {
		t.Fatal("duplicate request accepted")
	}
	if pipeline.Submit(request("overflow", "c.go")) {
		t.Fatal("overflow request accepted")
	}
	pipeline.Close()
	stats := pipeline.Stats()
	if stats.Accepted != 2 || stats.Duplicates != 1 || stats.Dropped != 1 {
		t.Fatalf("stats = %+v", stats)
	}
}

func TestPipelineFailureReturnsUnknown(t *testing.T) {
	pipeline := NewPipeline(2, 2, &fakeGitRunner{err: errors.New("git unavailable")})
	defer pipeline.Close()
	pipeline.Submit(request("failure", "a.go"))
	result := receive(t, pipeline)
	if result.Status != StatusUnknown || result.Source != SourceUnknown || result.Confidence != protocol.ConfidenceInferred {
		t.Fatalf("failure result = %+v", result)
	}
}

func TestPipelineDeduplicatesCompletedKeys(t *testing.T) {
	pipeline := NewPipeline(2, 2, nil)
	defer pipeline.Close()
	item := request("completed", "a.go")
	item.StructuredAdded, item.StructuredDeleted = number(1), number(0)
	if !pipeline.Submit(item) {
		t.Fatal("initial request rejected")
	}
	receive(t, pipeline)
	if pipeline.Submit(item) {
		t.Fatal("completed duplicate accepted")
	}
}

func TestCommandGitRunnerReadsNumstat(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "aav@example.invalid")
	runGit(t, root, "config", "user.name", "AAV Test")
	path := filepath.Join(root, "sample.txt")
	if err := os.WriteFile(path, []byte("one\ntwo\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "add", "sample.txt")
	runGit(t, root, "commit", "-m", "base")
	if err := os.WriteFile(path, []byte("one\nthree\nfour\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	values, err := NewCommandGitRunner(time.Second).Numstat(context.Background(), root, []string{"sample.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if values["sample.txt"].Added != 2 || values["sample.txt"].Deleted != 1 {
		t.Fatalf("numstat = %+v", values)
	}
}

func runGit(t *testing.T, root string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", root}, args...)...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}
