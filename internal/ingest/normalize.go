package ingest

import (
	"errors"
	"fmt"
	"path/filepath"

	adapter "github.com/greadee/agent-action-visualizer/adapter/go"
	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

func Normalize(event protocol.Event, selectedRoot string) (protocol.Event, error) {
	if err := event.Validate(); err != nil {
		return protocol.Event{}, err
	}
	if event.Path == "" && event.PreviousPath == "" {
		return event, nil
	}
	root, err := filepath.Abs(selectedRoot)
	if err != nil || selectedRoot == "" {
		return protocol.Event{}, errors.New("selected project root is required for path events")
	}
	event.ProjectRoot = root
	if event.Path, err = NormalizeProjectPath(root, event.Path); err != nil {
		return protocol.Event{}, fmt.Errorf("path: %w", err)
	}
	if event.PreviousPath != "" {
		if event.PreviousPath, err = NormalizeProjectPath(root, event.PreviousPath); err != nil {
			return protocol.Event{}, fmt.Errorf("previous_path: %w", err)
		}
	}
	return event, nil
}

// NormalizeProjectPath converts an absolute or project-relative path to a
// slash-separated project-relative path and rejects paths outside the root.
func NormalizeProjectPath(root, value string) (string, error) {
	return adapter.NormalizeProjectPath(root, value)
}
