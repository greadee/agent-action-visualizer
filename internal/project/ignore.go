package project

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

var defaultIgnorePatterns = []string{
	".git",
	".aav",
	"node_modules",
	"vendor",
	"dist",
	"build",
	"coverage",
	".next",
	"target",
	"__pycache__",
	".cache",
}

// IgnoreMatcher applies the same repository exclusions to scans and live
// filesystem observation. It intentionally preserves the scanner's existing
// subset of ignore syntax.
type IgnoreMatcher struct {
	root     string
	defaults []string
	patterns []string
}

func NewIgnoreMatcher(root string) *IgnoreMatcher {
	return newIgnoreMatcher(root, defaultIgnorePatterns)
}

func newIgnoreMatcher(root string, defaults []string) *IgnoreMatcher {
	matcher := &IgnoreMatcher{
		root:     root,
		defaults: append([]string(nil), defaults...),
	}
	matcher.Reload()
	return matcher
}

func (m *IgnoreMatcher) Reload() {
	if m == nil {
		return
	}
	patterns := append([]string(nil), m.defaults...)
	patterns = append(patterns, readPatterns(filepath.Join(m.root, ".gitignore"))...)
	patterns = append(patterns, readPatterns(filepath.Join(m.root, ".aavignore"))...)
	m.patterns = patterns
}

func (m *IgnoreMatcher) Ignored(path string, isDir bool) bool {
	if m == nil {
		return false
	}
	return ignored(filepath.ToSlash(path), isDir, m.patterns)
}

func readPatterns(path string) []string {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()
	var out []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "!") {
			out = append(out, strings.TrimSuffix(filepath.ToSlash(line), "/"))
		}
	}
	return out
}

func ignored(path string, _ bool, patterns []string) bool {
	parts := strings.Split(path, "/")
	for _, pattern := range patterns {
		pattern = strings.TrimPrefix(pattern, "/")
		if pattern == "" {
			continue
		}
		if !strings.Contains(pattern, "/") {
			for _, part := range parts {
				if match(pattern, part) {
					return true
				}
			}
		} else if match(pattern, path) || strings.HasPrefix(path, pattern+"/") {
			return true
		}
	}
	return false
}

func match(pattern, value string) bool {
	ok, _ := filepath.Match(filepath.FromSlash(pattern), filepath.FromSlash(value))
	return ok
}
