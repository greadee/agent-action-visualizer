package project

import (
	"bufio"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type NodeKind string

const (
	KindRoot          NodeKind = "root"
	KindDirectory     NodeKind = "directory"
	KindSource        NodeKind = "source"
	KindTest          NodeKind = "test"
	KindConfig        NodeKind = "config"
	KindDocumentation NodeKind = "documentation"
	KindAsset         NodeKind = "asset"
	KindGenerated     NodeKind = "generated"
)

type Node struct {
	ID         string       `json:"id,omitempty"`
	Path       string       `json:"path"`
	ParentPath string       `json:"parent_path,omitempty"`
	Name       string       `json:"name"`
	Kind       NodeKind     `json:"kind"`
	Size       int64        `json:"size,omitempty"`
	ModTime    int64        `json:"mod_time,omitempty"`
	Activity   NodeActivity `json:"activity,omitempty"`
}

type NodeActivity struct {
	GroupKey       string   `json:"group_key,omitempty"`
	LastCommit     string   `json:"last_commit,omitempty"`
	LastCommitAt   int64    `json:"last_commit_at,omitempty"`
	LastEvent      string   `json:"last_event,omitempty"`
	AccessCount    int      `json:"access_count,omitempty"`
	TotalTimeMS    int64    `json:"total_time_ms,omitempty"`
	LinesAdded     int64    `json:"lines_added,omitempty"`
	LinesDeleted   int64    `json:"lines_deleted,omitempty"`
	RecentTools    []string `json:"recent_tools,omitempty"`
	SessionHistory []string `json:"session_history,omitempty"`
	Confidence     string   `json:"confidence,omitempty"`
}
type Snapshot struct {
	Root  string `json:"root"`
	Nodes []Node `json:"nodes"`
}
type Scanner struct{ defaults []string }

func NewScanner() *Scanner {
	return &Scanner{defaults: []string{".git", ".aav", "node_modules", "vendor", "dist", "build", "coverage", ".next", "target", "__pycache__", ".cache"}}
}

func (s *Scanner) Scan(ctx context.Context, root string) (Snapshot, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return Snapshot{}, err
	}
	patterns := append([]string(nil), s.defaults...)
	patterns = append(patterns, readPatterns(filepath.Join(abs, ".gitignore"))...)
	patterns = append(patterns, readPatterns(filepath.Join(abs, ".aavignore"))...)
	nodes := []Node{{Path: ".", Name: filepath.Base(abs), Kind: KindRoot}}
	err = filepath.WalkDir(abs, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if path == abs {
			return nil
		}
		rel, err := filepath.Rel(abs, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if ignored(rel, entry.IsDir(), patterns) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		kind := classify(rel, entry.IsDir())
		parent := filepath.ToSlash(filepath.Dir(rel))
		if parent == "." {
			parent = "."
		}
		nodes = append(nodes, Node{Path: rel, ParentPath: parent, Name: entry.Name(), Kind: kind, Size: info.Size(), ModTime: info.ModTime().UnixNano()})
		return nil
	})
	if err != nil {
		return Snapshot{}, err
	}
	sort.Slice(nodes[1:], func(i, j int) bool { return nodes[i+1].Path < nodes[j+1].Path })
	return Snapshot{Root: abs, Nodes: nodes}, nil
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
func ignored(path string, isDir bool, patterns []string) bool {
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
func classify(path string, isDir bool) NodeKind {
	if isDir {
		return KindDirectory
	}
	lower := strings.ToLower(path)
	base := filepath.Base(lower)
	ext := filepath.Ext(lower)
	if strings.Contains(lower, "generated") || strings.HasSuffix(base, ".gen.go") {
		return KindGenerated
	}
	if strings.Contains(base, "test") || strings.Contains(lower, "/tests/") {
		return KindTest
	}
	switch ext {
	case ".md", ".mdx", ".rst", ".txt":
		return KindDocumentation
	case ".json", ".yaml", ".yml", ".toml", ".ini", ".env":
		return KindConfig
	case ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico", ".woff", ".woff2", ".mp4":
		return KindAsset
	default:
		return KindSource
	}
}
