package graph

import (
	"encoding/binary"
	"hash/fnv"
	"math"
	"sort"
	"strings"

	"github.com/greadee/agent-action-visualizer/internal/project"
)

type Vec3 struct{ X, Y, Z float64 }
type PositionedNode struct {
	project.Node
	Position Vec3 `json:"position"`
}
type Edge struct{ Source, Target string }
type GraphSnapshot struct {
	Revision int64
	Nodes    []PositionedNode
	Edges    []Edge
}

func Layout(nodes []project.Node, revision int64, previous map[string]Vec3) GraphSnapshot {
	ordered := append([]project.Node(nil), nodes...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Path < ordered[j].Path })
	positioned := make([]PositionedNode, 0, len(ordered))
	byPath := make(map[string]string, len(ordered))
	for _, node := range ordered {
		byPath[node.Path] = node.ID
		position := Vec3{}
		if node.Kind != project.KindRoot {
			if old, ok := previous[node.ID]; ok {
				position = old
			} else {
				position = positionFor(node.Path)
			}
		}
		positioned = append(positioned, PositionedNode{Node: node, Position: position})
	}
	edges := make([]Edge, 0, len(nodes))
	for _, node := range ordered {
		if node.Kind == project.KindRoot {
			continue
		}
		if parent, ok := byPath[node.ParentPath]; ok {
			edges = append(edges, Edge{Source: parent, Target: node.ID})
		}
	}
	return GraphSnapshot{Revision: revision, Nodes: positioned, Edges: edges}
}

func positionFor(path string) Vec3 {
	depth := len(strings.Split(path, "/"))
	radius := 2.6 + float64(depth)*2.4
	h := fnv.New64a()
	h.Write([]byte(path))
	sum := h.Sum(nil)
	a := float64(binary.BigEndian.Uint32(sum[:4])) / float64(math.MaxUint32)
	b := float64(binary.BigEndian.Uint32(sum[4:])) / float64(math.MaxUint32)
	theta := 2 * math.Pi * a
	z := 2*b - 1
	ring := math.Sqrt(math.Max(0, 1-z*z))
	return Vec3{X: radius * ring * math.Cos(theta), Y: radius * z, Z: radius * ring * math.Sin(theta)}
}

type Patch struct {
	Revision int64
	Added    []PositionedNode
	Updated  []PositionedNode
	Removed  []string
	Edges    []Edge
}

func Diff(previous, current GraphSnapshot) Patch {
	before := map[string]PositionedNode{}
	after := map[string]PositionedNode{}
	for _, n := range previous.Nodes {
		before[n.ID] = n
	}
	for _, n := range current.Nodes {
		after[n.ID] = n
	}
	patch := Patch{Revision: current.Revision, Edges: append([]Edge(nil), current.Edges...)}
	for id, node := range after {
		old, ok := before[id]
		if !ok {
			patch.Added = append(patch.Added, node)
		} else if old.Path != node.Path || old.Kind != node.Kind || old.Position != node.Position {
			patch.Updated = append(patch.Updated, node)
		}
	}
	for id := range before {
		if _, ok := after[id]; !ok {
			patch.Removed = append(patch.Removed, id)
		}
	}
	sort.Slice(patch.Added, func(i, j int) bool { return patch.Added[i].ID < patch.Added[j].ID })
	sort.Slice(patch.Updated, func(i, j int) bool { return patch.Updated[i].ID < patch.Updated[j].ID })
	sort.Strings(patch.Removed)
	return patch
}
