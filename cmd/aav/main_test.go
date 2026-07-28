package main

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodexStatusUsesIsolatedProject(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run(context.Background(), []string{
		"codex", "status", "--scope", "project", "--project", t.TempDir(),
	}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit = %d, stderr = %q", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "status: not-installed") ||
		!strings.Contains(stdout.String(), "managed hooks: 0") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestCodexInstallDryRunDoesNotWrite(t *testing.T) {
	root := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run(context.Background(), []string{
		"codex", "install", "--project", root, "--hook-binary", filepath.Join(root, "missing"), "--dry-run",
	}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 1 || !strings.Contains(stderr.String(), "hook binary does not exist") {
		t.Fatalf("exit = %d, stdout = %q, stderr = %q", exitCode, stdout.String(), stderr.String())
	}
}

func TestCodexUsageRejectsUnsupportedArguments(t *testing.T) {
	for _, arguments := range [][]string{
		{"codex"},
		{"codex", "unknown"},
		{"codex", "status", "--dry-run"},
		{"codex", "status", "extra"},
	} {
		var stderr bytes.Buffer
		if exitCode := run(context.Background(), arguments, strings.NewReader(""), &bytes.Buffer{}, &stderr); exitCode != 2 {
			t.Fatalf("args = %v, exit = %d, stderr = %q", arguments, exitCode, stderr.String())
		}
	}
}
