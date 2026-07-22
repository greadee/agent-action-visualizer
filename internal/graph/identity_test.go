package graph

import (
	"github.com/greadee/agent-action-visualizer/internal/project"
	"testing"
)

func TestIdentityDeterminismRenameAndTombstone(t *testing.T) {
	nodes := []project.Node{{Path: "src/a.go", Name: "a.go", Kind: project.KindSource}}
	a := NewIdentityRegistry("p", false).Assign(nodes)[0]
	b := NewIdentityRegistry("p", false).Assign(nodes)[0]
	if a.ID != b.ID {
		t.Fatal("identity is not deterministic")
	}
	registry := NewIdentityRegistry("p", false)
	original := registry.Assign(nodes)[0]
	renamed, ok := registry.Rename("src/a.go", "src/b.go")
	if !ok || renamed.ID != original.ID || registry.Aliases(original.ID)[0] != "src/a.go" {
		t.Fatalf("rename=%+v aliases=%v", renamed, registry.Aliases(original.ID))
	}
	deleted, ok := registry.Delete("src/b.go")
	if !ok || deleted.Kind != "tombstone" {
		t.Fatalf("delete=%+v", deleted)
	}
}
func TestCaseFoldAndSeparators(t *testing.T) {
	registry := NewIdentityRegistry("p", true)
	first := registry.Assign([]project.Node{{Path: "SRC\\A.go", Kind: project.KindSource}})[0]
	second := registry.Assign([]project.Node{{Path: "src/a.go", Kind: project.KindSource}})[0]
	if first.ID != second.ID {
		t.Fatalf("case/separator identity mismatch: %s %s", first.ID, second.ID)
	}
}
