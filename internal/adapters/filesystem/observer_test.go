package filesystem

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/greadee/agent-action-visualizer/adapter/go/wrapper"
	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

func TestObserverCoalescesWritesAndBatchesGitEvidence(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "file.go")
	if err := os.WriteFile(path, []byte("before\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	source := newFakeSource(64)
	git := &recordingGitInspector{evidence: GitEvidence{
		Status: map[string]GitStatus{"file.go": {Index: ' ', Worktree: 'M'}},
		Deltas: map[string]GitDelta{"file.go": {Added: 3, Deleted: 1}},
	}}
	observer := NewObserver(Config{
		Debounce: 10 * time.Millisecond,
		Git:      git,
		newSource: func(int) (eventSource, error) {
			return source, nil
		},
	})
	emitter := &captureEmitter{}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		observer.Observe(ctx, testObservation(root), emitter)
	}()
	source.waitReady(t)

	if err := os.WriteFile(path, []byte("after\nwith\nlines\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for range 20 {
		source.events <- change{path: path, op: opWrite}
	}
	event := emitter.waitFor(t, protocol.EventFileModified)
	cancel()
	<-done

	if event.Metadata["coalesced_events"] != 20 || event.LinesAdded == nil || *event.LinesAdded != 3 ||
		event.LinesDeleted == nil || *event.LinesDeleted != 1 {
		t.Fatalf("event = %#v", event)
	}
	if git.callCount() != 1 || len(git.lastPaths()) != 1 {
		t.Fatalf("git calls=%d paths=%#v", git.callCount(), git.lastPaths())
	}
}

func TestObserverCorrelatesUniqueRenameOutOfOrder(t *testing.T) {
	root := t.TempDir()
	oldPath := filepath.Join(root, "old.go")
	newPath := filepath.Join(root, "new.go")
	if err := os.WriteFile(oldPath, []byte("same bytes\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	fixedTime := time.Unix(1_700_000_000, 0)
	if err := os.Chtimes(oldPath, fixedTime, fixedTime); err != nil {
		t.Fatal(err)
	}
	source := newFakeSource(16)
	ready := make(chan struct{})
	observer := NewObserver(Config{
		Debounce:     5 * time.Millisecond,
		RenameWindow: 20 * time.Millisecond,
		Git:          &recordingGitInspector{},
		onReady:      func() { close(ready) },
		newSource: func(int) (eventSource, error) {
			return source, nil
		},
	})
	emitter := &captureEmitter{}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		observer.Observe(ctx, testObservation(root), emitter)
	}()
	select {
	case <-ready:
	case <-time.After(time.Second):
		t.Fatal("observer did not finish its initial snapshot")
	}

	if err := os.Rename(oldPath, newPath); err != nil {
		t.Fatal(err)
	}
	source.events <- change{path: newPath, op: opCreate}
	source.events <- change{path: oldPath, op: opRename}
	event := emitter.waitFor(t, protocol.EventFileRenamed)
	cancel()
	<-done

	if event.PreviousPath != oldPath || event.Path != newPath ||
		event.SourceConfidence != protocol.ConfidenceCorrelated {
		t.Fatalf("rename = %#v", event)
	}
}

func TestReadyChangesKeepsRenameCandidatesInOneBatch(t *testing.T) {
	started := time.Unix(1_700_000_000, 0)
	pending := map[string]*pendingChange{
		"new.go": {
			path:  "new.go",
			op:    opCreate,
			first: started,
			last:  started,
		},
		"old.go": {
			path:   "old.go",
			op:     opRename,
			before: fileState{exists: true},
			first:  started.Add(time.Millisecond),
			last:   started.Add(time.Millisecond),
		},
	}

	if ready := readyChanges(pending, started.Add(20*time.Millisecond), 5*time.Millisecond, 20*time.Millisecond, false, 16); len(ready) != 0 {
		t.Fatalf("rename candidates split before the shared window elapsed: %#v", ready)
	}
	if ready := readyChanges(pending, started.Add(21*time.Millisecond), 5*time.Millisecond, 20*time.Millisecond, false, 16); len(ready) != 2 {
		t.Fatalf("got %d rename candidates after the shared window, want 2", len(ready))
	}
}

func TestObserverIgnoresRulesAndReportsBurstOverload(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".aavignore"), []byte("ignored\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	source := newFakeSource(512)
	observer := NewObserver(Config{
		QueueSize:  2,
		MaxPending: 2,
		BatchSize:  2,
		Debounce:   100 * time.Millisecond,
		Git:        &recordingGitInspector{},
		newSource: func(int) (eventSource, error) {
			return source, nil
		},
	})
	emitter := &captureEmitter{}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		observer.Observe(ctx, testObservation(root), emitter)
	}()
	source.waitReady(t)

	ignored := filepath.Join(root, "ignored", "secret.txt")
	if err := os.MkdirAll(filepath.Dir(ignored), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ignored, []byte("not observed"), 0o600); err != nil {
		t.Fatal(err)
	}
	source.events <- change{path: ignored, op: opCreate}
	for index := range 200 {
		path := filepath.Join(root, "burst", time.Unix(0, int64(index)).Format("150405.000000000"))
		source.events <- change{path: path, op: opCreate}
	}
	emitter.waitFor(t, protocol.EventDroppedOrCoalesced)
	cancel()
	<-done

	events := emitter.events()
	var warning bool
	for _, event := range events {
		if event.Path == ignored {
			t.Fatalf("ignored path emitted: %#v", event)
		}
		if event.EventType == protocol.EventDroppedOrCoalesced {
			warning = true
		}
	}
	if !warning {
		t.Fatalf("overload warning missing from %#v", events)
	}
}

func TestObserverLongSessionKeepsProducingBoundedBatches(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "long.go")
	if err := os.WriteFile(path, []byte("0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	source := newFakeSource(128)
	git := &recordingGitInspector{}
	observer := NewObserver(Config{
		Debounce: 2 * time.Millisecond,
		Git:      git,
		newSource: func(int) (eventSource, error) {
			return source, nil
		},
	})
	emitter := &captureEmitter{}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		observer.Observe(ctx, testObservation(root), emitter)
	}()
	source.waitReady(t)

	const rounds = 50
	for index := range rounds {
		if err := os.WriteFile(path, []byte(time.Unix(int64(index), 0).String()), 0o600); err != nil {
			t.Fatal(err)
		}
		source.events <- change{path: path, op: opWrite}
		emitter.waitForCount(t, index+1)
	}
	cancel()
	<-done

	events := emitter.events()
	if len(events) != rounds || git.callCount() != rounds {
		t.Fatalf("events=%d git calls=%d", len(events), git.callCount())
	}
	for index, event := range events {
		if event.EventType != protocol.EventFileModified ||
			index > 0 && event.EventID <= events[index-1].EventID {
			t.Fatalf("event %d = %#v", index, event)
		}
	}
}

func TestObserverRealFilesystemSource(t *testing.T) {
	root := t.TempDir()
	ready := make(chan struct{})
	observer := NewObserver(Config{
		Debounce: 15 * time.Millisecond,
		Git:      &recordingGitInspector{},
		onReady:  func() { close(ready) },
		newSource: func(size int) (eventSource, error) {
			return newFSNotifySource(size)
		},
	})
	emitter := &captureEmitter{}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		observer.Observe(ctx, testObservation(root), emitter)
	}()
	select {
	case <-ready:
	case <-time.After(2 * time.Second):
		t.Fatal("real watcher did not start")
	}

	path := filepath.Join(root, "real.txt")
	directory := filepath.Join(root, "nested")
	if err := os.Mkdir(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	emitter.waitForPath(t, directory, protocol.EventDirectoryCreated)
	path = filepath.Join(directory, "real.txt")
	if err := os.WriteFile(path, []byte("observed"), 0o600); err != nil {
		t.Fatal(err)
	}
	event := emitter.waitForPath(t, path, protocol.EventFileCreated, protocol.EventFileModified)
	cancel()
	<-done
	if event.Path != path {
		t.Fatalf("real event = %#v", event)
	}
}

func testObservation(root string) wrapper.Observation {
	return wrapper.Observation{
		SessionID:   "filesystem-test",
		ProjectRoot: root,
		AgentType:   "test",
		ProcessID:   42,
	}
}

type fakeSource struct {
	events chan change
	errors chan error
	ready  chan struct{}
	once   sync.Once
}

func newFakeSource(capacity int) *fakeSource {
	return &fakeSource{
		events: make(chan change, capacity),
		errors: make(chan error, 1),
		ready:  make(chan struct{}),
	}
}

func (s *fakeSource) Add(string) error {
	s.once.Do(func() { close(s.ready) })
	return nil
}

func (s *fakeSource) Close() error          { return nil }
func (s *fakeSource) Events() <-chan change { return s.events }
func (s *fakeSource) Errors() <-chan error  { return s.errors }

func (s *fakeSource) waitReady(t *testing.T) {
	t.Helper()
	select {
	case <-s.ready:
	case <-time.After(time.Second):
		t.Fatal("observer did not start")
	}
}

type recordingGitInspector struct {
	mu       sync.Mutex
	evidence GitEvidence
	calls    int
	paths    []string
}

func (g *recordingGitInspector) Inspect(_ context.Context, _ string, paths []string) (GitEvidence, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.calls++
	g.paths = append([]string(nil), paths...)
	return g.evidence, nil
}

func (g *recordingGitInspector) callCount() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.calls
}

func (g *recordingGitInspector) lastPaths() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return append([]string(nil), g.paths...)
}

type captureEmitter struct {
	mu     sync.Mutex
	values []protocol.Event
}

func (e *captureEmitter) Emit(_ context.Context, events []protocol.Event) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.values = append(e.values, events...)
}

func (e *captureEmitter) events() []protocol.Event {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]protocol.Event(nil), e.values...)
}

func (e *captureEmitter) waitFor(t *testing.T, eventType protocol.EventType) protocol.Event {
	t.Helper()
	return e.waitForAny(t, eventType)
}

func (e *captureEmitter) waitForAny(t *testing.T, eventTypes ...protocol.EventType) protocol.Event {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		for _, event := range e.events() {
			for _, eventType := range eventTypes {
				if event.EventType == eventType {
					return event
				}
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("events not observed: %#v", e.events())
	return protocol.Event{}
}

func (e *captureEmitter) waitForPath(t *testing.T, path string, eventTypes ...protocol.EventType) protocol.Event {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		for _, event := range e.events() {
			if event.Path != path {
				continue
			}
			for _, eventType := range eventTypes {
				if event.EventType == eventType {
					return event
				}
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("path %q not observed: %#v", path, e.events())
	return protocol.Event{}
}

func (e *captureEmitter) waitForCount(t *testing.T, count int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if len(e.events()) >= count {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("got %d events, want %d", len(e.events()), count)
}
