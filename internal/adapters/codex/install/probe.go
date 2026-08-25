package install

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/greadee/agent-action-visualizer/internal/ipc"
	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

func probeInstalledHook(parent context.Context, binaryPath string) error {
	ctx, cancel := context.WithTimeout(parent, 3*time.Second)
	defer cancel()
	endpoint, err := diagnosticEndpoint()
	if err != nil {
		return err
	}
	received := make(chan protocol.Event, 1)
	server := ipc.NewServer(endpoint, func(event protocol.Event) bool {
		select {
		case received <- event:
		default:
		}
		return true
	})
	if err := server.Start(); err != nil {
		return fmt.Errorf("start isolated collector: %w", err)
	}
	defer server.Close()
	input, err := json.Marshal(map[string]string{
		"session_id": "aav-installer-test", "cwd": filepath.Dir(binaryPath),
		"hook_event_name": "SessionStart", "source": "startup",
	})
	if err != nil {
		return err
	}
	command := exec.CommandContext(ctx, binaryPath, "--aav-install-id", installID)
	command.Stdin = bytes.NewReader(input)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	command.Env = append(os.Environ(), "AAV_COLLECTOR_ENDPOINT="+endpoint)
	if err := command.Run(); err != nil {
		return fmt.Errorf("installed hook exited unsuccessfully: %w", err)
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		return errors.New("installed hook wrote unexpected output")
	}
	select {
	case event := <-received:
		if event.EventType != protocol.EventSessionStarted {
			return fmt.Errorf("installed hook emitted unexpected event type %q", event.EventType)
		}
		return nil
	case <-ctx.Done():
		return errors.New("installed hook did not reach the isolated collector")
	}
}

func diagnosticEndpoint() (string, error) {
	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	suffix := hex.EncodeToString(random)
	if runtime.GOOS == "windows" {
		return `\\.\pipe\aav-install-test-` + suffix, nil
	}
	return filepath.Join(os.TempDir(), "aav-install-test-"+suffix+".sock"), nil
}
