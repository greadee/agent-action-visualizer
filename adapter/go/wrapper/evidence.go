package wrapper

import (
	"context"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	adapter "github.com/greadee/agent-action-visualizer/adapter/go"
	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

const (
	DefaultEvidenceQueue  = 128
	DefaultEvidenceWindow = 100 * time.Millisecond
)

type evidenceGate struct {
	target   adapter.Emitter
	window   time.Duration
	capacity int
	queue    chan []protocol.Event
	stopCh   chan chan struct{}
	done     chan struct{}
	stopped  atomic.Bool
	once     sync.Once
}

type pendingEvidence struct {
	event   protocol.Event
	expires time.Time
}

func newEvidenceGate(target adapter.Emitter, capacity int, window time.Duration) *evidenceGate {
	if capacity <= 0 {
		capacity = DefaultEvidenceQueue
	}
	if window <= 0 {
		window = DefaultEvidenceWindow
	}
	gate := &evidenceGate{
		target:   target,
		window:   window,
		capacity: capacity,
		queue:    make(chan []protocol.Event, capacity),
		stopCh:   make(chan chan struct{}),
		done:     make(chan struct{}),
	}
	go gate.run()
	return gate
}

func (g *evidenceGate) Emit(ctx context.Context, events []protocol.Event) {
	if g == nil || g.stopped.Load() || len(events) == 0 {
		return
	}
	if ctx != nil {
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
	batch := append([]protocol.Event(nil), events...)
	select {
	case g.queue <- batch:
	default:
		// Strong evidence should not be displaced by queued fallback evidence.
		strong := make([]protocol.Event, 0, len(batch))
		for _, event := range batch {
			if !isFallbackEvidence(event) {
				strong = append(strong, event)
			}
		}
		g.target.Emit(context.Background(), strong)
	}
}

func (g *evidenceGate) run() {
	defer close(g.done)
	ticker := time.NewTicker(max(10*time.Millisecond, g.window/2))
	defer ticker.Stop()
	pending := make(map[string]pendingEvidence)
	recentStrong := make(map[string]time.Time)
	process := func(batch []protocol.Event, now time.Time) {
		for _, event := range batch {
			key := evidenceKey(event)
			if key == "" {
				g.target.Emit(context.Background(), []protocol.Event{event})
				continue
			}
			if isFallbackEvidence(event) {
				if seen, ok := recentStrong[key]; ok && now.Sub(seen) <= g.window {
					continue
				}
				if len(pending) >= g.capacity {
					if _, exists := pending[key]; !exists {
						continue
					}
				}
				pending[key] = pendingEvidence{event: event, expires: now.Add(g.window)}
				continue
			}
			delete(pending, key)
			recentStrong[key] = now
			g.target.Emit(context.Background(), []protocol.Event{event})
		}
	}
	for {
		select {
		case batch := <-g.queue:
			process(batch, time.Now())
		case now := <-ticker.C:
			flushEvidence(g.target, pending, now, false)
			for key, seen := range recentStrong {
				if now.Sub(seen) > 2*g.window {
					delete(recentStrong, key)
				}
			}
		case ack := <-g.stopCh:
			draining := true
			for draining {
				select {
				case batch := <-g.queue:
					process(batch, time.Now())
				default:
					draining = false
				}
			}
			flushEvidence(g.target, pending, time.Now(), true)
			close(ack)
			return
		}
	}
}

func flushEvidence(target adapter.Emitter, pending map[string]pendingEvidence, now time.Time, all bool) {
	keys := make([]string, 0, len(pending))
	for key, item := range pending {
		if all || !now.Before(item.expires) {
			keys = append(keys, key)
		}
	}
	sortStrings(keys)
	if len(keys) == 0 {
		return
	}
	events := make([]protocol.Event, 0, len(keys))
	for _, key := range keys {
		events = append(events, pending[key].event)
		delete(pending, key)
	}
	target.Emit(context.Background(), events)
}

func (g *evidenceGate) stop(timeout time.Duration) {
	if g == nil {
		return
	}
	g.once.Do(func() {
		g.stopped.Store(true)
		ack := make(chan struct{})
		select {
		case g.stopCh <- ack:
			select {
			case <-ack:
			case <-time.After(timeout):
			}
		case <-time.After(timeout):
		}
	})
	select {
	case <-g.done:
	case <-time.After(timeout):
	}
}

func evidenceKey(event protocol.Event) string {
	family := eventFamily(event.EventType)
	if family == "" || event.Path == "" {
		return ""
	}
	return strings.Join([]string{
		event.SessionID,
		family,
		canonicalEvidencePath(event.ProjectRoot, event.Path),
		canonicalEvidencePath(event.ProjectRoot, event.PreviousPath),
	}, "\x00")
}

func eventFamily(eventType protocol.EventType) string {
	switch eventType {
	case protocol.EventFileModified, protocol.EventFilePatched:
		return "file_write"
	case protocol.EventFileCreated:
		return "file_create"
	case protocol.EventFileDeleted:
		return "file_delete"
	case protocol.EventFileRenamed, protocol.EventFileMoved:
		return "file_move"
	case protocol.EventDirectoryCreated:
		return "directory_create"
	case protocol.EventDirectoryDeleted:
		return "directory_delete"
	case protocol.EventDirectoryRenamed:
		return "directory_move"
	default:
		return ""
	}
}

func canonicalEvidencePath(root, path string) string {
	if path == "" {
		return ""
	}
	if !filepath.IsAbs(path) && root != "" {
		path = filepath.Join(root, path)
	}
	path = filepath.Clean(path)
	if runtime.GOOS == "windows" {
		path = strings.ToLower(path)
	}
	return filepath.ToSlash(path)
}

func isFallbackEvidence(event protocol.Event) bool {
	return event.SourceType == protocol.SourceFilesystem || event.SourceType == protocol.SourceGit
}

func sortStrings(values []string) {
	for index := 1; index < len(values); index++ {
		for cursor := index; cursor > 0 && values[cursor] < values[cursor-1]; cursor-- {
			values[cursor], values[cursor-1] = values[cursor-1], values[cursor]
		}
	}
}

var _ adapter.Emitter = (*evidenceGate)(nil)
