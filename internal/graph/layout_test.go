package graph

import (
	"github.com/greadee/agent-action-visualizer/internal/project"
	"math"
	"reflect"
	"testing"
)

func graphNodes() []project.Node {
	return []project.Node{{ID: "root", Path: ".", Kind: project.KindRoot}, {ID: "src", Path: "src", ParentPath: ".", Kind: project.KindDirectory, Activity: project.NodeActivity{GroupKey: "commit-a"}}, {ID: "a", Path: "src/a.go", ParentPath: "src", Kind: project.KindSource, Activity: project.NodeActivity{GroupKey: "commit-a"}}}
}

func TestLayoutUsesUniformShellAndCommitClusters(t *testing.T) {
	nodes := append(graphNodes(),
		project.Node{ID: "b", Path: "src/b.go", ParentPath: "src", Kind: project.KindSource, Activity: project.NodeActivity{GroupKey: "commit-a"}},
		project.Node{ID: "c", Path: "docs/c.md", ParentPath: ".", Kind: project.KindDocumentation, Activity: project.NodeActivity{GroupKey: "commit-b"}},
	)
	laidOut := Layout(nodes, 1)
	positions := map[string]Vec3{}
	for _, node := range laidOut.Nodes {
		positions[node.ID] = node.Position
		if node.ID != "root" && math.Abs(length(node.Position)-shellRadius) > 0.000001 {
			t.Fatalf("node %s radius=%f", node.ID, length(node.Position))
		}
	}
	if distance(positions["a"], positions["b"]) >= distance(positions["a"], positions["c"]) {
		t.Fatalf("same-commit nodes should cluster: a-b=%f a-c=%f", distance(positions["a"], positions["b"]), distance(positions["a"], positions["c"]))
	}
}

func distance(a, b Vec3) float64 { return length(Vec3{X: a.X - b.X, Y: a.Y - b.Y, Z: a.Z - b.Z}) }
func TestLayoutDeterministicAndStableAcrossAddition(t *testing.T) {
	first := Layout(graphNodes(), 1)
	second := Layout(graphNodes(), 1)
	if !reflect.DeepEqual(first, second) {
		t.Fatal("layout changed for identical input")
	}
	positions := map[string]Vec3{}
	for _, n := range first.Nodes {
		positions[n.ID] = n.Position
	}
	expanded := append(graphNodes(), project.Node{ID: "b", Path: "src/b.go", ParentPath: "src", Kind: project.KindSource, Activity: project.NodeActivity{GroupKey: "commit-a"}})
	third := Layout(expanded, 2)
	for _, n := range third.Nodes {
		if old, ok := positions[n.ID]; ok && n.Position != old {
			t.Fatalf("existing position moved: %s", n.ID)
		}
	}
	if len(third.Edges) != 3 {
		t.Fatalf("edges=%+v", third.Edges)
	}
}
func TestGraphPatch(t *testing.T) {
	before := Layout(graphNodes(), 1)
	afterNodes := graphNodes()
	afterNodes[2].Path = "src/renamed.go"
	after := Layout(afterNodes, 2)
	patch := Diff(before, after)
	if len(patch.Updated) != 1 || patch.Updated[0].ID != "a" {
		t.Fatalf("patch=%+v", patch)
	}
}
