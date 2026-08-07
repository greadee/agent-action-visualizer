package ingest

import (
	"errors"
	"fmt"
	"path/filepath"

	adapter "github.com/greadee/agent-action-visualizer/adapter/go"
	"github.com/greadee/agent-action-visualizer/internal/security"
	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

func Normalize(event protocol.Event, selectedRoot string) (protocol.Event, error) {
	if err := event.Validate(); err != nil {
		return protocol.Event{}, err
	}
	event = security.SanitizeEvent(event)
	secondaryPaths, _ := event.Metadata["secondary_paths"].([]string)
	if event.Path == "" && event.PreviousPath == "" && len(secondaryPaths) == 0 {
		return event, nil
	}
	root, err := filepath.Abs(selectedRoot)
	if err != nil || selectedRoot == "" {
		return protocol.Event{}, errors.New("selected project root is required for path events")
	}
	event.ProjectRoot = root
	if event.Path != "" {
		if event.Path, err = NormalizeProjectPath(root, event.Path); err != nil {
			return protocol.Event{}, fmt.Errorf("path: %w", err)
		}
	}
	if event.PreviousPath != "" {
		if event.PreviousPath, err = NormalizeProjectPath(root, event.PreviousPath); err != nil {
			return protocol.Event{}, fmt.Errorf("previous_path: %w", err)
		}
	}
	if len(secondaryPaths) > 0 {
		normalized := make([]string, 0, len(secondaryPaths))
		for _, path := range secondaryPaths {
			value, pathErr := NormalizeProjectPath(root, path)
			if pathErr != nil {
				return protocol.Event{}, fmt.Errorf("secondary_paths: %w", pathErr)
			}
			normalized = append(normalized, value)
		}
		event.Metadata["secondary_paths"] = normalized
	}
	return event, nil
}

// NormalizeProjectPath converts an absolute or project-relative path to a
// slash-separated project-relative path and rejects paths outside the root.
func NormalizeProjectPath(root, value string) (string, error) {
	return adapter.NormalizeProjectPath(root, value)
}
