package project

import (
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
	return &Scanner{defaults: append([]string(nil), defaultIgnorePatterns...)}
}

func (s *Scanner) Scan(ctx context.Context, root string) (Snapshot, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return Snapshot{}, err
	}
	matcher := newIgnoreMatcher(abs, s.defaults)
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
		if matcher.Ignored(rel, entry.IsDir()) {
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
