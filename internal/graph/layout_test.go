package graph

import (
	"github.com/greadee/agent-action-visualizer/internal/project"
	"reflect"
	"testing"
)

func graphNodes() []project.Node {
	return []project.Node{{ID: "root", Path: ".", Kind: project.KindRoot}, {ID: "src", Path: "src", ParentPath: ".", Kind: project.KindDirectory}, {ID: "a", Path: "src/a.go", ParentPath: "src", Kind: project.KindSource}}
}
func TestLayoutDeterministicAndStableAcrossAddition(t *testing.T) {
	first := Layout(graphNodes(), 1, nil)
	second := Layout(graphNodes(), 1, nil)
	if !reflect.DeepEqual(first, second) {
		t.Fatal("layout changed for identical input")
	}
	positions := map[string]Vec3{}
	for _, n := range first.Nodes {
		positions[n.ID] = n.Position
	}
	expanded := append(graphNodes(), project.Node{ID: "b", Path: "src/b.go", ParentPath: "src", Kind: project.KindSource})
	third := Layout(expanded, 2, positions)
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
	before := Layout(graphNodes(), 1, nil)
	afterNodes := graphNodes()
	afterNodes[2].Path = "src/renamed.go"
	after := Layout(afterNodes, 2, map[string]Vec3{"a": before.Nodes[2].Position})
	patch := Diff(before, after)
	if len(patch.Updated) != 1 || patch.Updated[0].ID != "a" {
		t.Fatalf("patch=%+v", patch)
	}
}
