package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndRefreshProject(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main"), 0o600); err != nil {
		t.Fatal(err)
	}
	app := NewApp()
	initial, err := app.LoadProject(root)
	if err != nil {
		t.Fatal(err)
	}
	if initial.Revision != 1 || len(initial.Nodes) != 2 {
		t.Fatalf("unexpected initial snapshot: %#v", initial)
	}
	if err := os.WriteFile(filepath.Join(root, "main_test.go"), []byte("package main"), 0o600); err != nil {
		t.Fatal(err)
	}
	patch, err := app.RefreshProject()
	if err != nil {
		t.Fatal(err)
	}
	if patch.Revision != 2 || len(patch.Added) != 1 {
		t.Fatalf("unexpected refresh patch: %#v", patch)
	}
}
