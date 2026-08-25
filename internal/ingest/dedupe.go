package ingest

import (
	"sync"
	"time"

	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

type Deduper struct {
	mu     sync.Mutex
	window time.Duration
	seen   map[string]time.Time
}

func NewDeduper(window time.Duration) *Deduper {
	return &Deduper{window: window, seen: make(map[string]time.Time)}
}

func (d *Deduper) Duplicate(event protocol.Event, now time.Time) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	key := event.EventID
	if key == "" {
		key = string(event.EventType) + "\x00" + event.SessionID + "\x00" + event.Path + "\x00" + event.CorrelationID
	}
	if seen, ok := d.seen[key]; ok && now.Sub(seen) <= d.window {
		return true
	}
	d.seen[key] = now
	for candidate, seen := range d.seen {
		if now.Sub(seen) > d.window*2 {
			delete(d.seen, candidate)
		}
	}
	return false
}
