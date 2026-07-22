package session

import (
	"errors"
	"sort"
	"sync"
	"time"

	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

type AccessInterval struct {
	Sequence     int                 `json:"sequence"`
	NodeID       string              `json:"node_id,omitempty"`
	Path         string              `json:"path"`
	StartedAt    time.Time           `json:"started_at"`
	EndedAt      *time.Time          `json:"ended_at,omitempty"`
	Duration     time.Duration       `json:"duration"`
	Operations   []string            `json:"operations"`
	Source       protocol.SourceType `json:"source"`
	Confidence   protocol.Confidence `json:"confidence"`
	AgentID      string              `json:"agent_id,omitempty"`
	LinesAdded   *int64              `json:"lines_added,omitempty"`
	LinesDeleted *int64              `json:"lines_deleted,omitempty"`
}

type State struct {
	SessionID        string              `json:"session_id"`
	StartedAt        time.Time           `json:"started_at"`
	StoppedAt        *time.Time          `json:"stopped_at,omitempty"`
	Paused           bool                `json:"paused"`
	ActivePath       string              `json:"active_path,omitempty"`
	ActiveNodeID     string              `json:"active_node_id,omitempty"`
	PreviousPath     string              `json:"previous_path,omitempty"`
	PreviousNodeID   string              `json:"previous_node_id,omitempty"`
	SecondaryPaths   []string            `json:"secondary_paths,omitempty"`
	ActiveOperation  string              `json:"active_operation,omitempty"`
	ActiveSource     protocol.SourceType `json:"active_source,omitempty"`
	ActiveConfidence protocol.Confidence `json:"active_confidence,omitempty"`
	ActiveTimestamp  time.Time           `json:"active_timestamp,omitempty"`
	Intervals        []AccessInterval    `json:"intervals"`
	currentPriority  int
	currentTimestamp time.Time
}

type Engine struct {
	mu          sync.Mutex
	idleTimeout time.Duration
	sessions    map[string]*State
}

func NewEngine(idleTimeout time.Duration) *Engine {
	if idleTimeout <= 0 {
		idleTimeout = 2 * time.Minute
	}
	return &Engine{idleTimeout: idleTimeout, sessions: make(map[string]*State)}
}

func (e *Engine) Apply(event protocol.Event) (State, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if event.SessionID == "" {
		return State{}, errors.New("session_id is required")
	}
	s := e.sessions[event.SessionID]
	if event.EventType == protocol.EventSessionStarted {
		if s == nil {
			s = &State{SessionID: event.SessionID, StartedAt: event.Timestamp}
			e.sessions[event.SessionID] = s
		}
		return clone(s), nil
	}
	if s == nil {
		return State{}, errors.New("session has not started")
	}
	if event.EventType == protocol.EventSessionStopped {
		e.closeCurrent(s, event.Timestamp)
		stopped := event.Timestamp
		s.StoppedAt = &stopped
		return clone(s), nil
	}
	if s.StoppedAt != nil || s.Paused {
		return clone(s), nil
	}

	if event.EventType == protocol.EventFileRenamed || event.EventType == protocol.EventFileMoved {
		if s.ActivePath == event.PreviousPath && event.Path != "" {
			s.ActivePath = event.Path
			if current := current(s); current != nil {
				current.Path = event.Path
			}
		}
	}
	priority, active := focusPriority(event)
	if !active || event.Path == "" {
		return clone(s), nil
	}
	if event.Timestamp.Equal(s.currentTimestamp) && priority < s.currentPriority {
		return clone(s), nil
	}
	if s.ActivePath == event.Path {
		if current := current(s); current != nil {
			current.Operations = appendUnique(current.Operations, event.Operation)
			mergeDelta(current, event)
		}
		s.currentPriority, s.currentTimestamp = priority, event.Timestamp
		s.ActiveOperation, s.ActiveSource = event.Operation, event.SourceType
		s.ActiveConfidence, s.ActiveTimestamp = event.SourceConfidence, event.Timestamp
		s.SecondaryPaths = secondaryPaths(event.Metadata)
		return clone(s), nil
	}
	e.closeCurrent(s, event.Timestamp)
	s.PreviousPath, s.PreviousNodeID = s.ActivePath, s.ActiveNodeID
	s.ActivePath, s.ActiveNodeID = event.Path, event.NodeID
	s.SecondaryPaths = secondaryPaths(event.Metadata)
	s.ActiveOperation, s.ActiveSource = event.Operation, event.SourceType
	s.ActiveConfidence, s.ActiveTimestamp = event.SourceConfidence, event.Timestamp
	s.currentPriority, s.currentTimestamp = priority, event.Timestamp
	s.Intervals = append(s.Intervals, AccessInterval{Sequence: len(s.Intervals) + 1, NodeID: event.NodeID, Path: event.Path, StartedAt: event.Timestamp, Operations: appendUnique(nil, event.Operation), Source: event.SourceType, Confidence: event.SourceConfidence, AgentID: event.AgentID, LinesAdded: event.LinesAdded, LinesDeleted: event.LinesDeleted})
	return clone(s), nil
}

func (e *Engine) Pause(sessionID string, at time.Time) (State, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	s := e.sessions[sessionID]
	if s == nil {
		return State{}, errors.New("unknown session")
	}
	e.closeCurrent(s, at)
	s.Paused = true
	return clone(s), nil
}
func (e *Engine) Resume(sessionID string) (State, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	s := e.sessions[sessionID]
	if s == nil {
		return State{}, errors.New("unknown session")
	}
	s.Paused = false
	return clone(s), nil
}
func (e *Engine) Advance(sessionID string, now time.Time) (State, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	s := e.sessions[sessionID]
	if s == nil {
		return State{}, errors.New("unknown session")
	}
	if c := current(s); c != nil && c.EndedAt == nil && now.Sub(c.StartedAt) >= e.idleTimeout {
		e.closeCurrent(s, c.StartedAt.Add(e.idleTimeout))
	}
	return clone(s), nil
}

func (e *Engine) closeCurrent(s *State, at time.Time) {
	c := current(s)
	if c == nil || c.EndedAt != nil {
		return
	}
	limit := c.StartedAt.Add(e.idleTimeout)
	if at.After(limit) {
		at = limit
	}
	if at.Before(c.StartedAt) {
		at = c.StartedAt
	}
	c.EndedAt = &at
	c.Duration = at.Sub(c.StartedAt)
}
func current(s *State) *AccessInterval {
	if len(s.Intervals) == 0 {
		return nil
	}
	return &s.Intervals[len(s.Intervals)-1]
}

func focusPriority(event protocol.Event) (int, bool) {
	confidence := map[protocol.Confidence]int{protocol.ConfidenceInferred: 0, protocol.ConfidenceObserved: 10, protocol.ConfidenceCorrelated: 20, protocol.ConfidenceExact: 30}[event.SourceConfidence]
	operation := 0
	switch event.EventType {
	case protocol.EventFileCreated:
		operation = 9
	case protocol.EventFilePatched, protocol.EventFileModified:
		operation = 8
	case protocol.EventFileRenamed, protocol.EventFileMoved:
		operation = 7
	case protocol.EventFileDeleted:
		operation = 6
	case protocol.EventFileFocused:
		operation = 5
	case protocol.EventFileRead:
		operation = 3
	default:
		return 0, false
	}
	return confidence + operation, true
}

func appendUnique(values []string, value string) []string {
	if value == "" {
		return values
	}
	for _, v := range values {
		if v == value {
			return values
		}
	}
	return append(values, value)
}
func mergeDelta(interval *AccessInterval, event protocol.Event) {
	if event.LinesAdded != nil {
		interval.LinesAdded = event.LinesAdded
	}
	if event.LinesDeleted != nil {
		interval.LinesDeleted = event.LinesDeleted
	}
}
func secondaryPaths(metadata map[string]interface{}) []string {
	value, ok := metadata["secondary_paths"]
	if !ok {
		return nil
	}
	var out []string
	switch raw := value.(type) {
	case []string:
		out = append(out, raw...)
	case []interface{}:
		out = make([]string, 0, len(raw))
		for _, v := range raw {
			if path, ok := v.(string); ok && path != "" {
				out = append(out, path)
			}
		}
	}
	out = compactPaths(out)
	sort.Strings(out)
	return out
}

func compactPaths(paths []string) []string {
	out := make([]string, 0, len(paths))
	seen := make(map[string]bool, len(paths))
	for _, path := range paths {
		if path != "" && !seen[path] {
			seen[path] = true
			out = append(out, path)
		}
	}
	return out
}
func clone(s *State) State {
	copyState := *s
	copyState.SecondaryPaths = append([]string(nil), s.SecondaryPaths...)
	copyState.Intervals = append([]AccessInterval(nil), s.Intervals...)
	for i := range copyState.Intervals {
		copyState.Intervals[i].Operations = append([]string(nil), s.Intervals[i].Operations...)
	}
	return copyState
}
