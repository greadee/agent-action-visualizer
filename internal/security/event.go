// Package security centralizes local event minimization and display-safe labels.
package security

import (
	"math"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

const maxSecondaryPaths = 128

var (
	secretAssignment = regexp.MustCompile(`(?i)\b(password|passwd|secret|token|api[_-]?key|authorization|cookie)\s*[:=]\s*(?:"[^"]*"|'[^']*'|[^\s,;]+)`)
	bearerToken      = regexp.MustCompile(`(?i)\bbearer\s+[^\s,;]+`)
	knownToken       = regexp.MustCompile(`\b(?:gh[pousr]_[A-Za-z0-9_]{16,}|sk-[A-Za-z0-9_-]{16,}|AKIA[0-9A-Z]{16})\b`)
)

// SanitizeEvent removes fields that are not required by visualization and
// restricts metadata to the keys consumed by deterministic reducers.
func SanitizeEvent(event protocol.Event) protocol.Event {
	event.AgentType = Label(event.AgentType, 64)
	event.AdapterID = Label(event.AdapterID, 128)
	event.AdapterVersion = Label(event.AdapterVersion, 64)
	event.Operation = Label(event.Operation, 64)
	event.Status = Label(event.Status, 64)
	event.ToolName = Label(event.ToolName, 256)
	event.ActionLabel = Label(event.ActionLabel, 256)
	event.Command = ""
	event.Metadata = sanitizeMetadata(event.Metadata)
	return event
}

// CommandLabel derives a bounded display label without retaining arguments or
// control characters.
func CommandLabel(command string, limit int) string {
	label := Label(filepath.Base(strings.TrimSpace(command)), limit)
	if label == "" || label == "." {
		return "command"
	}
	return label
}

// Label produces single-line, valid UTF-8 text and redacts common secret forms.
func Label(value string, limit int) string {
	value = strings.ToValidUTF8(value, "")
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, value)
	value = strings.Join(strings.Fields(value), " ")
	value = secretAssignment.ReplaceAllStringFunc(value, func(match string) string {
		separator := strings.IndexAny(match, ":=")
		if separator < 0 {
			return "<redacted>"
		}
		return strings.TrimSpace(match[:separator]) + "=<redacted>"
	})
	value = bearerToken.ReplaceAllString(value, "Bearer <redacted>")
	value = knownToken.ReplaceAllString(value, "<redacted>")
	return truncateUTF8(value, limit)
}

func sanitizeMetadata(metadata map[string]interface{}) map[string]interface{} {
	if len(metadata) == 0 {
		return nil
	}
	out := make(map[string]interface{})
	for key, value := range metadata {
		switch key {
		case "access_sequence", "coalesced_events", "dropped_events", "exit_code":
			if integer, ok := metadataInteger(value); ok {
				out[key] = integer
			}
		case "evidence", "git_status", "hook_event", "reason", "signal", "work_confidence", "work_source":
			if text, ok := value.(string); ok {
				out[key] = Label(text, 128)
			}
		case "secondary_paths":
			if paths := metadataPaths(value); len(paths) > 0 {
				out[key] = paths
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func metadataInteger(value interface{}) (interface{}, bool) {
	switch typed := value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return typed, true
	case float32:
		if !math.IsNaN(float64(typed)) && !math.IsInf(float64(typed), 0) && typed == float32(math.Trunc(float64(typed))) {
			return typed, true
		}
	case float64:
		if !math.IsNaN(typed) && !math.IsInf(typed, 0) && typed == math.Trunc(typed) {
			return typed, true
		}
	}
	return nil, false
}

func metadataPaths(value interface{}) []string {
	var values []string
	switch typed := value.(type) {
	case []string:
		values = typed
	case []interface{}:
		for _, item := range typed {
			text, ok := item.(string)
			if !ok {
				return nil
			}
			values = append(values, text)
		}
	default:
		return nil
	}
	if len(values) > maxSecondaryPaths {
		values = values[:maxSecondaryPaths]
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(strings.ToValidUTF8(value, ""))
		if value != "" {
			out = append(out, truncateUTF8(value, 4096))
		}
	}
	return out
}

func truncateUTF8(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	value = value[:limit]
	for !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	return strings.TrimSpace(value)
}
