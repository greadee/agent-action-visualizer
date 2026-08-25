//go:build windows

package ipc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"os/user"

	"github.com/Microsoft/go-winio"
)

func platformDefaultEndpoint() string {
	identity := "default"
	if current, err := user.Current(); err == nil && current.Uid != "" {
		identity = current.Uid
	}
	sum := sha256.Sum256([]byte(identity))
	return `\\.\pipe\aav-collector-v1-` + hex.EncodeToString(sum[:6])
}

func listenLocal(endpoint string) (net.Listener, error) {
	descriptor := "D:P(A;;GA;;;SY)(A;;GA;;;OW)"
	if current, err := user.Current(); err == nil && current.Uid != "" {
		descriptor = fmt.Sprintf("D:P(A;;GA;;;SY)(A;;GA;;;%s)", current.Uid)
	}
	return winio.ListenPipe(endpoint, &winio.PipeConfig{
		SecurityDescriptor: descriptor,
		InputBufferSize:    MaxPayloadBytes,
		OutputBufferSize:   64,
	})
}

func dialLocal(ctx context.Context, endpoint string) (net.Conn, error) {
	return winio.DialPipeContext(ctx, endpoint)
}

func cleanupLocal(string) {}

func testEndpoint(seed string) string {
	sum := sha256.Sum256([]byte(seed))
	return `\\.\pipe\aav-test-` + hex.EncodeToString(sum[:8])
}
