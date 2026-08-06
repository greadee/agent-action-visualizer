package ingest

import (
	"context"
	"fmt"
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
