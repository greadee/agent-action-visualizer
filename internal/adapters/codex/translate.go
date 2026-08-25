package codex

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/greadee/agent-action-visualizer/internal/buildinfo"
	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

type fileEvidence struct {
	eventType    protocol.EventType
	operation    string
	path         string
	previousPath string
	linesAdded   *int64
	linesDeleted *int64
	isDirectory  *bool
}

func Translate(input HookInput, observedAt time.Time) []protocol.Event {
	base := protocol.Event{
		SchemaVersion:    protocol.SchemaVersion,
		SessionID:        limitText(input.SessionID, 128),
		ThreadID:         limitText(input.SessionID, 128),
		TurnID:           limitText(input.TurnID, 128),
		AgentID:          limitText(input.AgentID, 128),
		AgentType:        "codex",
		AdapterID:        "codex-hooks",
		AdapterVersion:   limitText(buildinfo.Version, 64),
		SourceType:       protocol.SourceNativeHook,
		SourceConfidence: protocol.ConfidenceExact,
		Timestamp:        observedAt.UTC(),
	}

	var events []protocol.Event
	switch input.HookEventName {
	case "SessionStart":
		if input.Source == "compact" {
			return nil
		}
		event := base
		event.EventType = protocol.EventSessionStarted
		event.Operation = "session"
		event.Status = limitText(input.Source, 64)
		events = append(events, event)
	case "SessionEnd":
		event := base
		event.EventType = protocol.EventSessionStopped
		event.Operation = "session"
		event.Status = limitText(input.Reason, 64)
		events = append(events, event)
	case "SubagentStart":
		event := base
		event.EventType = protocol.EventAgentStarted
		event.AgentType = limitText(input.AgentType, 64)
		event.Operation = "subagent"
		event.Status = "started"
		events = append(events, event)
	case "SubagentStop":
		event := base
		event.EventType = protocol.EventAgentStopped
		event.AgentType = limitText(input.AgentType, 64)
		event.Operation = "subagent"
		event.Status = "stopped"
		events = append(events, event)
	case "PreToolUse":
		events = append(events, toolEvent(base, input, protocol.EventToolStarted, "started"))
	case "PostToolUse":
		parent := toolEvent(base, input, protocol.EventToolCompleted, "completed")
		events = append(events, parent)
		for _, evidence := range extractFileEvidence(input) {
			if len(events) >= MaxEventsPerHook {
				break
			}
			event := base
			event.EventType = evidence.eventType
			event.Operation = evidence.operation
			event.Status = "reported"
			event.ToolName = limitText(input.ToolName, 256)
			event.CorrelationID = limitText(input.ToolUseID, 128)
			event.ParentEventID = parent.EventID
			event.SourceConfidence = protocol.ConfidenceCorrelated
			event.Path = resolvePath(input.CWD, evidence.path)
			event.PreviousPath = resolvePath(input.CWD, evidence.previousPath)
			event.LinesAdded = evidence.linesAdded
			event.LinesDeleted = evidence.linesDeleted
			event.IsDirectory = evidence.isDirectory
			if event.Path == "" || (event.EventType == protocol.EventFileMoved || event.EventType == protocol.EventFileRenamed) && event.PreviousPath == "" {
				continue
			}
			events = append(events, event)
		}
	default:
		return nil
	}

	for index := range events {
		events[index].EventID = eventID(events[index], input.HookEventName, index)
		if events[index].Metadata == nil {
			events[index].Metadata = map[string]interface{}{}
		}
		events[index].Metadata["hook_event"] = input.HookEventName
	}
	if input.HookEventName == "PostToolUse" && len(events) > 1 {
		for index := 1; index < len(events); index++ {
			events[index].ParentEventID = events[0].EventID
		}
	}
	return events
}

func toolEvent(base protocol.Event, input HookInput, eventType protocol.EventType, status string) protocol.Event {
	event := base
	event.EventType = eventType
	event.Operation = "tool"
	event.Status = status
	event.ToolName = limitText(input.ToolName, 256)
	event.CorrelationID = limitText(input.ToolUseID, 128)
	return event
}

func extractFileEvidence(input HookInput) []fileEvidence {
	if strings.EqualFold(input.ToolName, "apply_patch") {
		var values map[string]json.RawMessage
		if json.Unmarshal(input.ToolInput, &values) != nil {
			return nil
		}
		return parsePatch(rawString(values, "command"))
	}
	return extractStructuredPaths(input.ToolName, input.ToolInput)
}

func extractStructuredPaths(toolName string, raw json.RawMessage) []fileEvidence {
	var values map[string]json.RawMessage
	if len(raw) == 0 || json.Unmarshal(raw, &values) != nil {
		return nil
	}
	name := strings.ToLower(strings.ReplaceAll(toolName, "-", "_"))
	paths := rawStrings(values, "path", "file_path", "filepath", "paths", "file_paths")
	var eventType protocol.EventType
	var operation string
	switch {
	case toolToken(name, "read_file") || toolToken(name, "read_text_file") || toolToken(name, "view_image"):
		eventType, operation = protocol.EventFileRead, "read"
	case toolToken(name, "create_file"):
		eventType, operation = protocol.EventFileCreated, "create"
	case toolToken(name, "write_file") || toolToken(name, "write_text_file") || toolToken(name, "edit_file") || toolToken(name, "replace_in_file"):
		eventType, operation = protocol.EventFileModified, "modify"
	case toolToken(name, "delete_file") || toolToken(name, "remove_file"):
		eventType, operation = protocol.EventFileDeleted, "delete"
	case toolToken(name, "create_directory"):
		eventType, operation = protocol.EventDirectoryCreated, "create"
	case toolToken(name, "delete_directory") || toolToken(name, "remove_directory"):
		eventType, operation = protocol.EventDirectoryDeleted, "delete"
	case toolToken(name, "rename_file") || toolToken(name, "move_file"):
		oldPath := firstRawString(values, "source", "source_path", "old_path", "from")
		newPath := firstRawString(values, "destination", "destination_path", "new_path", "to")
		if oldPath == "" || newPath == "" {
			return nil
		}
		kind, op := moveKind(oldPath, newPath)
		return []fileEvidence{{eventType: kind, operation: op, path: newPath, previousPath: oldPath}}
	default:
		return nil
	}
	result := make([]fileEvidence, 0, len(paths))
	for _, path := range paths {
		if safePath(path) == "" || len(result) >= MaxEventsPerHook-1 {
			continue
		}
		var isDirectory *bool
		if eventType == protocol.EventDirectoryCreated || eventType == protocol.EventDirectoryDeleted {
			isDirectory = boolPointer(true)
		}
		result = append(result, fileEvidence{eventType: eventType, operation: operation, path: path, isDirectory: isDirectory})
	}
	return result
}

func toolToken(name, token string) bool {
	return name == token || strings.HasSuffix(name, "__"+token) || strings.HasSuffix(name, "_"+token)
}

func rawStrings(values map[string]json.RawMessage, keys ...string) []string {
	result := make([]string, 0, 4)
	seen := map[string]bool{}
	for _, key := range keys {
		raw, ok := values[key]
		if !ok {
			continue
		}
		var single string
		if json.Unmarshal(raw, &single) == nil {
			if single = safePath(single); single != "" && !seen[single] {
				seen[single] = true
				result = append(result, single)
			}
			continue
		}
		var many []string
		if json.Unmarshal(raw, &many) == nil {
			for _, value := range many {
				value = safePath(value)
				if value != "" && !seen[value] && len(result) < MaxEventsPerHook-1 {
					seen[value] = true
					result = append(result, value)
				}
			}
		}
	}
	return result
}

func rawString(values map[string]json.RawMessage, key string) string {
	var value string
	_ = json.Unmarshal(values[key], &value)
	return value
}

func firstRawString(values map[string]json.RawMessage, keys ...string) string {
	for _, key := range keys {
		if value := safePath(rawString(values, key)); value != "" {
			return value
		}
	}
	return ""
}

func resolvePath(cwd, value string) string {
	value = safePath(value)
	if value == "" {
		return ""
	}
	if filepath.IsAbs(value) || cwd == "" || strings.IndexByte(cwd, 0) >= 0 {
		return limitText(filepath.Clean(value), 4096)
	}
	return limitText(filepath.Clean(filepath.Join(cwd, value)), 4096)
}

func safePath(value string) string {
	value = strings.TrimSpace(strings.ToValidUTF8(value, ""))
	if value == "" || strings.IndexByte(value, 0) >= 0 || len(value) > 4096 {
		return ""
	}
	return value
}

func eventID(event protocol.Event, hookName string, index int) string {
	parts := []string{event.SessionID, event.TurnID, event.AgentID, event.CorrelationID, hookName, string(event.EventType), event.Operation, event.Status, event.ToolName, event.Path, event.PreviousPath, string(rune(index))}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return "codex:" + hex.EncodeToString(sum[:16])
}

func limitText(value string, size int) string {
	value = strings.ToValidUTF8(value, "")
	if len(value) <= size {
		return value
	}
	value = value[:size]
	for !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	return value
}

func boolPointer(value bool) *bool { return &value }
