package adapter

import (
	"errors"
	"path/filepath"
	"strings"
)

// NormalizeProjectPath converts an absolute or project-relative path to a
// slash-separated project-relative path and rejects paths outside root.
func NormalizeProjectPath(root, value string) (string, error) {
	if root == "" {
		return "", errors.New("project root is required")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
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
