//go:build !windows

package wrapper_test

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/greadee/agent-action-visualizer/adapter/go/wrapper"
)

func TestUnixSignalIsForwardedAndPreserved(t *testing.T) {
	if os.Getenv("AAV_SIGNAL_WRAPPER_HELPER") == "1" {
		result := wrapper.Run(context.Background(), wrapper.Config{
			Command: os.Args[0],
			Args:    []string{"-test.run=TestUnixSignalChildHelper"},
			Env:     setEnvironment(os.Environ(), "AAV_SIGNAL_CHILD_HELPER", "1"),
			Stdout:  os.Stdout,
			Stderr:  os.Stderr,
		})
		wrapper.Exit(result)
	}

	command := exec.Command(os.Args[0], "-test.run=TestUnixSignalIsForwardedAndPreserved")
	command.Env = setEnvironment(os.Environ(), "AAV_SIGNAL_WRAPPER_HELPER", "1")
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	command.Stderr = os.Stderr
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	ready := make(chan string, 1)
	go func() {
		line, _ := bufio.NewReader(stdout).ReadString('\n')
		ready <- line
	}()
	select {
	case line := <-ready:
		if line != "ready\n" {
			t.Fatalf("child readiness = %q", line)
		}
	case <-time.After(time.Second):
		_ = command.Process.Kill()
		t.Fatal("timed out waiting for child readiness")
	}
	if err := command.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	err = command.Wait()
	exitError, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("wrapper wait error = %v", err)
	}
	status, ok := exitError.ProcessState.Sys().(syscall.WaitStatus)
	if !ok || !status.Signaled() || status.Signal() != syscall.SIGTERM {
		t.Fatalf("wrapper status = %#v", exitError.ProcessState.Sys())
	}
}

func TestUnixSignalChildHelper(t *testing.T) {
	if os.Getenv("AAV_SIGNAL_CHILD_HELPER") != "1" {
		return
	}
	_, _ = fmt.Fprintln(os.Stdout, "ready")
	time.Sleep(10 * time.Second)
}
