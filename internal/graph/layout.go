package graph

import (
	"encoding/binary"
	"hash/fnv"
	"math"
	"reflect"
	"sort"

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

const shellRadius = 7.0

func Layout(nodes []project.Node, revision int64) GraphSnapshot {
	ordered := append([]project.Node(nil), nodes...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Path < ordered[j].Path })
	centers := groupCenters(ordered)
	positioned := make([]PositionedNode, 0, len(ordered))
	byPath := make(map[string]string, len(ordered))
	for _, node := range ordered {
		byPath[node.Path] = node.ID
		position := Vec3{}
		if node.Kind != project.KindRoot {
			position = positionFor(node, centers[groupKey(node)])
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

func positionFor(node project.Node, center Vec3) Vec3 {
	u := cross(center, Vec3{X: 0, Y: 1, Z: 0})
	if length(u) < 0.1 {
		u = cross(center, Vec3{X: 1, Y: 0, Z: 0})
	}
	u = normalize(u)
	v := normalize(cross(center, u))
	a, b := hashPair("node:" + node.Path)
	angle := 2 * math.Pi * a
	spread := 0.06 + 0.2*math.Sqrt(b)
	direction := add(center, scale(add(scale(u, math.Cos(angle)), scale(v, math.Sin(angle))), spread))
	return scale(normalize(direction), shellRadius)
}

func groupKey(node project.Node) string {
	if node.Activity.GroupKey != "" {
		return node.Activity.GroupKey
	}
	return "uncommitted:" + node.ParentPath
}

func groupCenters(nodes []project.Node) map[string]Vec3 {
	type group struct {
		key       string
		timestamp int64
	}
	byKey := map[string]int64{}
	for _, node := range nodes {
		if node.Kind == project.KindRoot {
			continue
		}
		key := groupKey(node)
		current, exists := byKey[key]
		if !exists || node.Activity.LastCommitAt > current {
			byKey[key] = node.Activity.LastCommitAt
		}
	}
	groups := make([]group, 0, len(byKey))
	for key, timestamp := range byKey {
		groups = append(groups, group{key: key, timestamp: timestamp})
	}
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].timestamp != groups[j].timestamp {
			return groups[i].timestamp > groups[j].timestamp
		}
		return groups[i].key < groups[j].key
	})
	centers := make(map[string]Vec3, len(groups))
	goldenAngle := math.Pi * (3 - math.Sqrt(5))
	for index, group := range groups {
		y := 1 - 2*(float64(index)+0.5)/float64(len(groups))
		ring := math.Sqrt(math.Max(0, 1-y*y))
		theta := goldenAngle * float64(index)
		centers[group.key] = Vec3{X: ring * math.Cos(theta), Y: y, Z: ring * math.Sin(theta)}
	}
	return centers
}

func hashPair(value string) (float64, float64) {
	h := fnv.New64a()
	_, _ = h.Write([]byte(value))
	sum := h.Sum(nil)
	return float64(binary.BigEndian.Uint32(sum[:4])) / float64(math.MaxUint32), float64(binary.BigEndian.Uint32(sum[4:])) / float64(math.MaxUint32)
}

func add(a, b Vec3) Vec3 { return Vec3{X: a.X + b.X, Y: a.Y + b.Y, Z: a.Z + b.Z} }
func scale(v Vec3, factor float64) Vec3 {
	return Vec3{X: v.X * factor, Y: v.Y * factor, Z: v.Z * factor}
}
func cross(a, b Vec3) Vec3 {
	return Vec3{X: a.Y*b.Z - a.Z*b.Y, Y: a.Z*b.X - a.X*b.Z, Z: a.X*b.Y - a.Y*b.X}
}
func length(v Vec3) float64 { return math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z) }
func normalize(v Vec3) Vec3 {
	l := length(v)
	if l == 0 {
		return Vec3{Z: 1}
	}
	return scale(v, 1/l)
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
		} else if old.Path != node.Path || old.Kind != node.Kind || old.Position != node.Position || !reflect.DeepEqual(old.Activity, node.Activity) {
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
