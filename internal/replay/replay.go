// Package replay rebuilds deterministic session state from persisted events.
package replay

import (
	"errors"
	"time"

	workdiff "github.com/greadee/agent-action-visualizer/internal/diff"
	"github.com/greadee/agent-action-visualizer/internal/session"
	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

// Snapshot is a stable, cursor-bounded session view. Cursor is -1 for an
// empty pre-session view; otherwise it indexes Events in persistence order.
type Snapshot struct {
	State      session.State
	Cursor     int
	CursorAt   time.Time
	EventCount int
}

// Reconstruct applies events through cursor. An unfinished access is closed at
// the cursor timestamp so replay geometry never depends on wall-clock time.
func Reconstruct(events []protocol.Event, cursor int, idleTimeout time.Duration) (Snapshot, error) {
	if len(events) == 0 || cursor < 0 {
		return Snapshot{Cursor: -1, EventCount: len(events)}, nil
	}
	if cursor >= len(events) {
		cursor = len(events) - 1
	}
	engine := session.NewEngine(idleTimeout)
	var state session.State
	var cursorAt time.Time
	for index := 0; index <= cursor; index++ {
		event := events[index]
		if event.Timestamp.After(cursorAt) {
			cursorAt = event.Timestamp
		}
		var err error
		if event.EventType == protocol.EventDiffCalculated {
			state, err = applyDiff(engine, event)
		} else {
			state, err = engine.Apply(event)
		}
		if err != nil {
			return Snapshot{}, err
		}
	}
	if state.SessionID == "" {
		return Snapshot{}, errors.New("session start is missing from persisted events")
	}
	if state.StoppedAt == nil && !cursorAt.IsZero() {
		state, _ = engine.Pause(state.SessionID, cursorAt)
	}
	return Snapshot{State: state, Cursor: cursor, CursorAt: cursorAt, EventCount: len(events)}, nil
}

func applyDiff(engine *session.Engine, event protocol.Event) (session.State, error) {
	sequence, ok := metadataInt(event.Metadata, "access_sequence")
	if !ok || sequence < 1 {
		return session.State{}, errors.New("diff result access_sequence is required")
	}
	status := workdiff.Status(event.Status)
	source := workdiff.Source(event.Operation)
	if status == "" {
		status = workdiff.StatusUnknown
	}
	if source == "" {
		source = workdiff.SourceUnknown
	}
	return engine.SetWorkResult(event.SessionID, sequence, event.LinesAdded, event.LinesDeleted, status, source, event.SourceConfidence)
}

func metadataInt(metadata map[string]interface{}, key string) (int, bool) {
	value, ok := metadata[key]
	if !ok {
		return 0, false
	}
	switch value := value.(type) {
	case int:
		return value, true
	case float64:
		return int(value), float64(int(value)) == value
	default:
		return 0, false
	}
}
