package ingest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

func event(id string, kind protocol.EventType, path string) protocol.Event {
	return protocol.Event{SchemaVersion: protocol.SchemaVersion, EventID: id, EventType: kind, SourceType: protocol.SourceNativeHook, SourceConfidence: protocol.ConfidenceExact, Timestamp: time.Now(), Path: path}
}

func TestQueueCoalescesReadsAndProtectsWrites(t *testing.T) {
	q := NewQueue(2)
	q.Offer(event("r1", protocol.EventFileRead, "a.go"))
	q.Offer(event("r2", protocol.EventFileRead, "a.go"))
	if q.Len() != 1 || q.Stats().Coalesced != 1 {
		t.Fatalf("read was not coalesced: len=%d stats=%+v", q.Len(), q.Stats())
	}
	q.Offer(event("r3", protocol.EventFileRead, "b.go"))
	if !q.Offer(event("w1", protocol.EventFilePatched, "c.go")) {
		t.Fatal("high-value write was dropped")
	}
	if q.Len() != 2 || q.Stats().Dropped != 1 {
		t.Fatalf("unexpected overload state: len=%d stats=%+v", q.Len(), q.Stats())
	}
}

func TestNormalizeRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	valid, err := Normalize(event("1", protocol.EventFileRead, filepath.Join(root, "src", "a.go")), root)
	if err != nil || valid.Path != "src/a.go" {
		t.Fatalf("valid path: %#v %v", valid, err)
	}
	if _, err := Normalize(event("2", protocol.EventFileRead, filepath.Join(root, "..", "secret")), root); err == nil {
		t.Fatal("out-of-root path accepted")
	}
	secondary := event("3", protocol.EventFileRead, filepath.Join(root, "safe.go"))
	secondary.Metadata = map[string]interface{}{"secondary_paths": []string{"../secret"}}
	if _, err := Normalize(secondary, root); err == nil {
		t.Fatal("out-of-root secondary path accepted")
	}
}

func TestNormalizeRejectsPrimaryAndSecondarySymlinkEscapes(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(root, "outside-link")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	primary := event("primary", protocol.EventFileRead, filepath.Join(link, "secret.txt"))
	if _, err := Normalize(primary, root); err == nil {
		t.Fatal("primary symlink escape was accepted")
	}
	secondary := event("secondary", protocol.EventFileRead, filepath.Join(root, "safe.txt"))
	secondary.Metadata = map[string]interface{}{"secondary_paths": []string{filepath.Join(link, "secret.txt")}}
	if _, err := Normalize(secondary, root); err == nil {
		t.Fatal("secondary symlink escape was accepted")
	}
}

func TestNormalizeMinimizesMetadataAndCommand(t *testing.T) {
	root := t.TempDir()
	e := event("sanitize", protocol.EventFileRead, filepath.Join(root, "safe.txt"))
	e.Command = "tool --token private-value"
	e.ActionLabel = "read\npassword=private-value"
	e.Metadata = map[string]interface{}{
		"access_sequence": 7,
		"prompt":          "private-value",
		"tool_response":   "private-value",
		"environment":     map[string]interface{}{"TOKEN": "private-value"},
	}
	normalized, err := Normalize(e, root)
	if err != nil {
		t.Fatal(err)
	}
	if normalized.Command != "" || normalized.ActionLabel == e.ActionLabel {
		t.Fatalf("command or label was not sanitized: %#v", normalized)
	}
	if len(normalized.Metadata) != 1 || normalized.Metadata["access_sequence"] != 7 {
		t.Fatalf("metadata was not minimized: %#v", normalized.Metadata)
	}
}

func TestDeduperWindow(t *testing.T) {
	d := NewDeduper(time.Second)
	now := time.Now()
	e := event("same", protocol.EventFileRead, "a.go")
	if d.Duplicate(e, now) || !d.Duplicate(e, now.Add(time.Millisecond)) || d.Duplicate(e, now.Add(2*time.Second)) {
		t.Fatal("unexpected duplicate classification")
	}
}

func TestCollectorProcessesAsynchronously(t *testing.T) {
	processed := make(chan string, 1)
	collector := NewCollector(4, func(_ context.Context, e protocol.Event) { processed <- e.EventID })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	collector.Start(ctx)
	defer collector.Stop()
	if !collector.Submit(event("async", protocol.EventFileRead, "a.go")) {
		t.Fatal("submit failed")
	}
	select {
	case id := <-processed:
		if id != "async" {
			t.Fatal(id)
		}
	case <-time.After(time.Second):
		t.Fatal("collector stalled")
	}
}

func TestCollectorStopDrainsAcceptedEvents(t *testing.T) {
	processed := make(chan string, 4)
	collector := NewCollector(4, func(_ context.Context, event protocol.Event) { processed <- event.EventID })
	collector.Start(context.Background())
	for _, id := range []string{"one", "two", "three"} {
		if !collector.Submit(event(id, protocol.EventFilePatched, id+".go")) {
			t.Fatalf("event %s was not accepted", id)
		}
	}
	collector.Stop()
	close(processed)
	var count int
	for range processed {
		count++
	}
	if count != 3 {
		t.Fatalf("processed %d accepted events", count)
	}
}

func TestQueueBurstRemainsBounded(t *testing.T) {
	const capacity = 256
	q := NewQueue(capacity)
	for index := 0; index < 100_000; index++ {
		kind := protocol.EventFileRead
		if index%1_000 == 0 {
			kind = protocol.EventFilePatched
		}
		q.Offer(event(fmt.Sprintf("burst-%d", index), kind, fmt.Sprintf("src/file-%d.go", index%2_048)))
		if q.Len() > capacity {
			t.Fatalf("queue exceeded capacity: %d", q.Len())
		}
	}
	if q.Len() != capacity {
		t.Fatalf("unexpected final queue length: %d", q.Len())
	}
	if q.Stats().Dropped == 0 {
		t.Fatal("overload did not report dropped events")
	}
}

func BenchmarkQueueOffer(b *testing.B) {
	q := NewQueue(1024)
	e := event("bench", protocol.EventFileRead, "a.go")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Offer(e)
	}
}

func BenchmarkQueueBurst(b *testing.B) {
	events := make([]protocol.Event, 4_096)
	for index := range events {
		events[index] = event(fmt.Sprintf("burst-%d", index), protocol.EventFileRead, fmt.Sprintf("src/file-%d.go", index))
	}
	q := NewQueue(256)
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		q.Offer(events[index%len(events)])
	}
}
