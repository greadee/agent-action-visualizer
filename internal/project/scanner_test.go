package project

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestScanIgnoresAndClassifies(t *testing.T) {
	root := t.TempDir()
	write := func(path, body string) {
		full := filepath.Join(root, path)
		os.MkdirAll(filepath.Dir(full), 0o755)
		if err := os.WriteFile(full, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(".gitignore", "ignored.txt\n")
	write(".aavignore", "private/\n")
	write("src/main.go", "package main")
	write("src/main_test.go", "package main")
	write("README.md", "docs")
	write("ignored.txt", "x")
	write("private/key.txt", "x")
	write("node_modules/pkg/a.js", "x")
	snapshot, err := NewScanner().Scan(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]NodeKind{}
	for _, node := range snapshot.Nodes {
		kinds[node.Path] = node.Kind
	}
	if kinds["src/main.go"] != KindSource || kinds["src/main_test.go"] != KindTest || kinds["README.md"] != KindDocumentation {
		t.Fatalf("classification: %+v", kinds)
	}
	for _, blocked := range []string{"ignored.txt", "private/key.txt", "node_modules/pkg/a.js"} {
		if _, ok := kinds[blocked]; ok {
			t.Fatalf("ignored path included: %s", blocked)
		}
	}
}
