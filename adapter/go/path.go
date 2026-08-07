package adapter

import (
	"errors"
	"fmt"
	"os"
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
	if err := rejectSymlinkEscape(root, abs); err != nil {
		return "", err
	}
	return filepath.ToSlash(filepath.Clean(rel)), nil
}

func rejectSymlinkEscape(root, candidate string) error {
	resolvedRoot := root
	rootInfo, err := os.Lstat(root)
	if err != nil {
		return fmt.Errorf("inspect selected project: %w", err)
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 {
		resolvedRoot, err = filepath.EvalSymlinks(root)
		if err != nil {
			return fmt.Errorf("resolve selected project: %w", err)
		}
	}
	relative, err := filepath.Rel(root, candidate)
	if err != nil {
		return err
	}
	current := root
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		if component == "" || component == "." {
			continue
		}
		current = filepath.Join(current, component)
		info, statErr := os.Lstat(current)
		if errors.Is(statErr, os.ErrNotExist) {
			return nil
		}
		if statErr != nil {
			return fmt.Errorf("inspect project path: %w", statErr)
		}
		if info.Mode()&os.ModeSymlink == 0 {
			continue
		}
		resolved, resolveErr := filepath.EvalSymlinks(current)
		if resolveErr != nil {
			return fmt.Errorf("resolve project path: %w", resolveErr)
		}
		if !pathWithin(resolvedRoot, resolved) {
			return errors.New("path escapes selected project through a symbolic link")
		}
	}
	return nil
}

func pathWithin(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}
