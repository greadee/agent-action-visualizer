//go:build !windows

package ipc

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUnixEndpointIsOwnerOnly(t *testing.T) {
	endpoint := filepath.Join(t.TempDir(), "collector.sock")
	server := NewServer(endpoint, nil)
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	info, err := os.Stat(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	if permissions := info.Mode().Perm(); permissions != 0o600 {
		t.Fatalf("socket permissions = %04o, want 0600", permissions)
	}
}
