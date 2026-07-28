//go:build !windows

package ipc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"
	"os"
	"path/filepath"
	"time"
)

func platformDefaultEndpoint() string {
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "agent-action-visualizer", "collector-v1.sock")
}

func listenLocal(endpoint string) (net.Listener, error) {
	if err := os.MkdirAll(filepath.Dir(endpoint), 0o700); err != nil {
		return nil, err
	}
	if info, err := os.Lstat(endpoint); err == nil && info.Mode()&os.ModeSocket != 0 {
		connection, dialErr := net.DialTimeout("unix", endpoint, 20*time.Millisecond)
		if dialErr == nil {
			_ = connection.Close()
			return nil, errors.New("collector endpoint is already in use")
		}
		_ = os.Remove(endpoint)
	}
	listener, err := net.Listen("unix", endpoint)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(endpoint, 0o600); err != nil {
		_ = listener.Close()
		_ = os.Remove(endpoint)
		return nil, err
	}
	return listener, nil
}

func dialLocal(ctx context.Context, endpoint string) (net.Conn, error) {
	return (&net.Dialer{}).DialContext(ctx, "unix", endpoint)
}

func cleanupLocal(endpoint string) { _ = os.Remove(endpoint) }

func testEndpoint(seed string) string {
	sum := sha256.Sum256([]byte(seed))
	return filepath.Join(os.TempDir(), "aav-"+hex.EncodeToString(sum[:6])+".sock")
}
