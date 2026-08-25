package ingest

import (
	"sync"

	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

type QueueStats struct{ Accepted, Dropped, Coalesced uint64 }

type Queue struct {
	mu       sync.Mutex
	capacity int
	items    []protocol.Event
	stats    QueueStats
}

func NewQueue(capacity int) *Queue {
	if capacity < 1 {
		capacity = 1
	}
	return &Queue{capacity: capacity, items: make([]protocol.Event, 0, capacity)}
}

func (q *Queue) Offer(event protocol.Event) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	if isCoalescible(event.EventType) {
		for i := len(q.items) - 1; i >= 0; i-- {
			queued := q.items[i]
			if queued.EventType == event.EventType && queued.Path == event.Path && queued.SessionID == event.SessionID {
				q.items[i] = event
				q.stats.Coalesced++
				return true
			}
		}
	}
	if len(q.items) == q.capacity {
		if !highValue(event.EventType) {
			q.stats.Dropped++
			return false
		}
		drop := -1
		for i, queued := range q.items {
			if !highValue(queued.EventType) {
				drop = i
				break
			}
		}
		if drop < 0 {
			q.stats.Dropped++
			return false
		}
		copy(q.items[drop:], q.items[drop+1:])
		q.items = q.items[:len(q.items)-1]
		q.stats.Dropped++
	}
	q.items = append(q.items, event)
	q.stats.Accepted++
	return true
}

func (q *Queue) Pop() (protocol.Event, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) == 0 {
		return protocol.Event{}, false
	}
	event := q.items[0]
	copy(q.items, q.items[1:])
	q.items = q.items[:len(q.items)-1]
	return event, true
}

func (q *Queue) Len() int          { q.mu.Lock(); defer q.mu.Unlock(); return len(q.items) }
func (q *Queue) Stats() QueueStats { q.mu.Lock(); defer q.mu.Unlock(); return q.stats }

func isCoalescible(t protocol.EventType) bool {
	return t == protocol.EventFileRead || t == protocol.EventFileFocused
}
func highValue(t protocol.EventType) bool {
	switch t {
	case protocol.EventFileCreated, protocol.EventFileModified, protocol.EventFilePatched, protocol.EventFileRenamed, protocol.EventFileMoved, protocol.EventFileDeleted,
		protocol.EventDirectoryCreated, protocol.EventDirectoryRenamed, protocol.EventDirectoryDeleted, protocol.EventSessionStarted, protocol.EventSessionStopped:
		return true
	default:
		return false
	}
}
