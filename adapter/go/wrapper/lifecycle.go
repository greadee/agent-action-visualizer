package wrapper

import (
	"path/filepath"
	"strings"
	"time"

	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

func lifecycleStarted(observation Observation, command string, at time.Time) []protocol.Event {
	label := commandLabel(command)
	sessionID := observation.SessionID + ":session-started"
	commandID := observation.SessionID + ":command-started"
	return []protocol.Event{
		baseEvent(observation, sessionID, protocol.EventSessionStarted, at, label),
		func() protocol.Event {
			event := baseEvent(observation, commandID, protocol.EventCommandStarted, at, label)
			event.Operation = "execute"
			event.Status = "running"
			event.ParentEventID = sessionID
			event.CorrelationID = observation.SessionID + ":command"
			return event
		}(),
	}
}

func lifecycleStopped(observation Observation, command string, startedAt, stoppedAt time.Time, result Result) []protocol.Event {
	label := commandLabel(command)
	duration := stoppedAt.Sub(startedAt).Milliseconds()
	if duration < 0 {
		duration = 0
	}
	status := "completed"
	if result.Signal != "" {
		status = "signaled"
	} else if result.ExitCode != 0 {
		status = "failed"
	}
	metadata := map[string]interface{}{"exit_code": result.ExitCode}
	if result.Signal != "" {
		metadata["signal"] = result.Signal
	}
	commandID := observation.SessionID + ":command-completed"
	sessionID := observation.SessionID + ":session-stopped"
	return []protocol.Event{
		func() protocol.Event {
			event := baseEvent(observation, commandID, protocol.EventCommandCompleted, stoppedAt, label)
			event.Operation = "execute"
			event.Status = status
			event.ParentEventID = observation.SessionID + ":command-started"
			event.CorrelationID = observation.SessionID + ":command"
			event.DurationMS = &duration
			event.Metadata = metadata
			return event
		}(),
		func() protocol.Event {
			event := baseEvent(observation, sessionID, protocol.EventSessionStopped, stoppedAt, label)
			event.Status = status
			event.ParentEventID = observation.SessionID + ":session-started"
			event.DurationMS = &duration
			event.Metadata = metadata
			return event
		}(),
	}
}

func baseEvent(observation Observation, eventID string, eventType protocol.EventType, at time.Time, label string) protocol.Event {
	return protocol.Event{
		SchemaVersion:    protocol.SchemaVersion,
		EventID:          eventID,
		SessionID:        observation.SessionID,
		AgentType:        observation.AgentType,
		AdapterID:        AdapterID,
		AdapterVersion:   AdapterVersion,
		SourceType:       protocol.SourceWrapper,
		SourceConfidence: protocol.ConfidenceExact,
		EventType:        eventType,
		Timestamp:        at.UTC(),
		ProcessID:        observation.ProcessID,
		ProjectRoot:      observation.ProjectRoot,
		ActionLabel:      label,
	}
}

func commandLabel(command string) string {
	label := strings.TrimSpace(filepath.Base(command))
	if label == "" || label == "." {
		return "command"
	}
	if len(label) > 128 {
		return label[:128]
	}
	return label
}
