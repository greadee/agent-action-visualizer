// Package protocol defines the agent-independent normalized event contract.
package protocol

import (
	"errors"
	"fmt"
	"time"
)

const SchemaVersion = "1.0"

type Confidence string

const (
	ConfidenceExact      Confidence = "exact"
	ConfidenceCorrelated Confidence = "correlated"
	ConfidenceObserved   Confidence = "observed"
	ConfidenceInferred   Confidence = "inferred"
)

type SourceType string

const (
	SourceNativeHook       SourceType = "native_hook"
	SourceStructuredStream SourceType = "structured_stream"
	SourceEditor           SourceType = "editor"
	SourceFilesystem       SourceType = "filesystem"
	SourceGit              SourceType = "git"
	SourceWrapper          SourceType = "wrapper"
	SourceSynthetic        SourceType = "synthetic"
)

type EventType string

const (
	EventSessionStarted       EventType = "session_started"
	EventSessionStopped       EventType = "session_stopped"
	EventAgentStarted         EventType = "agent_started"
	EventAgentStopped         EventType = "agent_stopped"
	EventToolStarted          EventType = "tool_started"
	EventToolCompleted        EventType = "tool_completed"
	EventFileFocused          EventType = "file_focused"
	EventFileRead             EventType = "file_read"
	EventFileCreated          EventType = "file_created"
	EventFileModified         EventType = "file_modified"
	EventFilePatched          EventType = "file_patched"
	EventFileRenamed          EventType = "file_renamed"
	EventFileMoved            EventType = "file_moved"
	EventFileDeleted          EventType = "file_deleted"
	EventDirectoryCreated     EventType = "directory_created"
	EventDirectoryRenamed     EventType = "directory_renamed"
	EventDirectoryDeleted     EventType = "directory_deleted"
	EventCommandStarted       EventType = "command_started"
	EventCommandCompleted     EventType = "command_completed"
	EventAccessIntervalOpened EventType = "access_interval_opened"
	EventAccessIntervalClosed EventType = "access_interval_closed"
	EventDiffCalculated       EventType = "diff_calculated"
	EventProjectRescanned     EventType = "project_rescanned"
	EventAdapterConnected     EventType = "adapter_connected"
	EventAdapterDisconnected  EventType = "adapter_disconnected"
	EventAdapterWarning       EventType = "adapter_warning"
	EventDroppedOrCoalesced   EventType = "event_dropped_or_coalesced"
)

type Event struct {
	SchemaVersion      string                 `json:"schema_version"`
	EventID            string                 `json:"event_id"`
	SessionID          string                 `json:"session_id,omitempty"`
	ProjectID          string                 `json:"project_id,omitempty"`
	AgentID            string                 `json:"agent_id,omitempty"`
	AgentType          string                 `json:"agent_type,omitempty"`
	AdapterID          string                 `json:"adapter_id,omitempty"`
	AdapterVersion     string                 `json:"adapter_version,omitempty"`
	SourceType         SourceType             `json:"source_type"`
	SourceConfidence   Confidence             `json:"source_confidence"`
	EventType          EventType              `json:"event_type"`
	Operation          string                 `json:"operation,omitempty"`
	Status             string                 `json:"status,omitempty"`
	Timestamp          time.Time              `json:"timestamp"`
	MonotonicTimestamp int64                  `json:"monotonic_timestamp,omitempty"`
	CorrelationID      string                 `json:"correlation_id,omitempty"`
	ParentEventID      string                 `json:"parent_event_id,omitempty"`
	ProcessID          int                    `json:"process_id,omitempty"`
	ThreadID           string                 `json:"thread_id,omitempty"`
	TurnID             string                 `json:"turn_id,omitempty"`
	ToolName           string                 `json:"tool_name,omitempty"`
	Command            string                 `json:"command,omitempty"`
	ProjectRoot        string                 `json:"project_root,omitempty"`
	Path               string                 `json:"path,omitempty"`
	PreviousPath       string                 `json:"previous_path,omitempty"`
	NodeID             string                 `json:"node_id,omitempty"`
	IsDirectory        *bool                  `json:"is_directory,omitempty"`
	IsBinary           *bool                  `json:"is_binary,omitempty"`
	DurationMS         *int64                 `json:"duration_ms,omitempty"`
	LinesAdded         *int64                 `json:"lines_added,omitempty"`
	LinesDeleted       *int64                 `json:"lines_deleted,omitempty"`
	BytesBefore        *int64                 `json:"bytes_before,omitempty"`
	BytesAfter         *int64                 `json:"bytes_after,omitempty"`
	ContentHashBefore  string                 `json:"content_hash_before,omitempty"`
	ContentHashAfter   string                 `json:"content_hash_after,omitempty"`
	PhaseID            string                 `json:"phase_id,omitempty"`
	SliceID            string                 `json:"slice_id,omitempty"`
	ActionLabel        string                 `json:"action_label,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}

func (e Event) Validate() error {
	if e.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported schema_version %q", e.SchemaVersion)
	}
	if e.EventID == "" {
		return errors.New("event_id is required")
	}
	if !validEventTypes[e.EventType] {
		return fmt.Errorf("unsupported event_type %q", e.EventType)
	}
	if !validSources[e.SourceType] {
		return fmt.Errorf("unsupported source_type %q", e.SourceType)
	}
	if !validConfidences[e.SourceConfidence] {
		return fmt.Errorf("unsupported source_confidence %q", e.SourceConfidence)
	}
	if e.Timestamp.IsZero() {
		return errors.New("timestamp is required")
	}
	for name, value := range map[string]*int64{"duration_ms": e.DurationMS, "lines_added": e.LinesAdded, "lines_deleted": e.LinesDeleted, "bytes_before": e.BytesBefore, "bytes_after": e.BytesAfter} {
		if value != nil && *value < 0 {
			return fmt.Errorf("%s must be non-negative", name)
		}
	}
	return nil
}

var validConfidences = map[Confidence]bool{ConfidenceExact: true, ConfidenceCorrelated: true, ConfidenceObserved: true, ConfidenceInferred: true}
var validSources = map[SourceType]bool{SourceNativeHook: true, SourceStructuredStream: true, SourceEditor: true, SourceFilesystem: true, SourceGit: true, SourceWrapper: true, SourceSynthetic: true}
var validEventTypes = map[EventType]bool{
	EventSessionStarted: true, EventSessionStopped: true, EventAgentStarted: true, EventAgentStopped: true,
	EventToolStarted: true, EventToolCompleted: true, EventFileFocused: true, EventFileRead: true,
	EventFileCreated: true, EventFileModified: true, EventFilePatched: true, EventFileRenamed: true,
	EventFileMoved: true, EventFileDeleted: true, EventDirectoryCreated: true, EventDirectoryRenamed: true,
	EventDirectoryDeleted: true, EventCommandStarted: true, EventCommandCompleted: true,
	EventAccessIntervalOpened: true, EventAccessIntervalClosed: true, EventDiffCalculated: true,
	EventProjectRescanned: true, EventAdapterConnected: true, EventAdapterDisconnected: true,
	EventAdapterWarning: true, EventDroppedOrCoalesced: true,
}
