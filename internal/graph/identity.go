package graph

import (
	"crypto/sha256"
	"encoding/hex"
	pathpkg "path"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/greadee/agent-action-visualizer/internal/project"
)

type IdentityRegistry struct {
	mu        sync.Mutex
	projectID string
	caseFold  bool
	byPath    map[string]project.Node
	aliases   map[string][]string
}

func NewIdentityRegistry(projectID string, caseFold bool) *IdentityRegistry {
	return &IdentityRegistry{projectID: projectID, caseFold: caseFold, byPath: make(map[string]project.Node), aliases: make(map[string][]string)}
}
func (r *IdentityRegistry) Assign(nodes []project.Node) []project.Node {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]project.Node, len(nodes))
	for i, node := range nodes {
		key := r.key(node.Path)
		if existing, ok := r.byPath[key]; ok {
			node.ID = existing.ID
		} else {
			node.ID = stableID(r.projectID, key)
		}
		r.byPath[key] = node
		out[i] = node
	}
	return out
}
func (r *IdentityRegistry) Rename(previous, next string) (project.Node, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	oldKey, newKey := r.key(previous), r.key(next)
	node, ok := r.byPath[oldKey]
	if !ok {
		return project.Node{}, false
	}
	delete(r.byPath, oldKey)
	r.aliases[node.ID] = append(r.aliases[node.ID], node.Path)
	node.Path = filepath.ToSlash(filepath.Clean(next))
	node.Name = filepath.Base(next)
	node.ParentPath = filepath.ToSlash(filepath.Dir(next))
	r.byPath[newKey] = node
	return node, true
}
func (r *IdentityRegistry) Delete(path string) (project.Node, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := r.key(path)
	node, ok := r.byPath[key]
	if !ok {
		return project.Node{}, false
	}
	node.Kind = "tombstone"
	r.byPath[key] = node
	return node, true
}

// Lookup returns the latest known node for a project-relative path. It lets
// live event handling retain a stable ID before the next scanner refresh.
func (r *IdentityRegistry) Lookup(path string) (project.Node, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	node, ok := r.byPath[r.key(path)]
	return node, ok
}

// Tombstones returns deleted nodes that remain inspectable until a later
// lifecycle policy prunes them. Results are sorted for deterministic refreshes.
func (r *IdentityRegistry) Tombstones() []project.Node {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]project.Node, 0)
	for _, node := range r.byPath {
		if node.Kind == "tombstone" {
			result = append(result, node)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result
}
func (r *IdentityRegistry) Aliases(id string) []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.aliases[id]...)
}
func (r *IdentityRegistry) key(path string) string {
	key := pathpkg.Clean(strings.ReplaceAll(path, "\\", "/"))
	if r.caseFold {
		key = strings.ToLower(key)
	}
	return key
}
func stableID(projectID, path string) string {
	sum := sha256.Sum256([]byte(projectID + "\x00" + path))
	return "node_" + hex.EncodeToString(sum[:12])
}
