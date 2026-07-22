package ingest

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

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
	if event.Path, err = relativeWithin(root, event.Path); err != nil {
		return protocol.Event{}, fmt.Errorf("path: %w", err)
	}
	if event.PreviousPath != "" {
		if event.PreviousPath, err = relativeWithin(root, event.PreviousPath); err != nil {
			return protocol.Event{}, fmt.Errorf("previous_path: %w", err)
		}
	}
	return event, nil
}

func relativeWithin(root, value string) (string, error) {
	candidate := value
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(root, candidate)
	}
	abs, err := filepath.Abs(candidate)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", errors.New("path escapes selected project")
	}
	return filepath.ToSlash(filepath.Clean(rel)), nil
}
