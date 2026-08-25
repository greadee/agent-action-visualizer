package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/greadee/agent-action-visualizer/adapter/go/wrapper"
)

func TestRunRejectsMissingCommand(t *testing.T) {
	var stderr bytes.Buffer
	_, usageCode := run(context.Background(), nil, strings.NewReader(""), &bytes.Buffer{}, &stderr)
	if usageCode != 2 || !strings.Contains(stderr.String(), "usage: aav-wrapper") {
		t.Fatalf("usage=%d stderr=%q", usageCode, stderr.String())
	}
}

func TestWrapperProcessPreservesExitAndOutput(t *testing.T) {
	command := exec.Command(os.Args[0], "-test.run=TestWrapperMainHelperProcess")
	command.Env = append(os.Environ(), "AAV_WRAPPER_MAIN_HELPER=1", "AAV_WRAPPED_CHILD_HELPER=1")
	command.Stdin = strings.NewReader("process input")
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	exitError, ok := err.(*exec.ExitError)
	if !ok || exitError.ExitCode() != 23 {
		t.Fatalf("process err=%v stdout=%q stderr=%q", err, stdout.String(), stderr.String())
	}
	if stdout.String() != "stdout:process input\n" || stderr.String() != "stderr:process input\n" {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestWrapperMainHelperProcess(t *testing.T) {
	if os.Getenv("AAV_WRAPPER_MAIN_HELPER") != "1" {
		return
	}
	endpoint := filepath.Join(t.TempDir(), "missing.sock")
	if runtime.GOOS == "windows" {
		endpoint = `\\.\pipe\aav-wrapper-missing`
	}
	result, usageCode := run(context.Background(), []string{
		"--endpoint", endpoint,
		"--session-id", "process-session",
		"--",
		os.Args[0], "-test.run=TestWrappedChildHelperProcess",
	}, os.Stdin, os.Stdout, os.Stderr)
	if usageCode != 0 {
		os.Exit(usageCode)
	}
	wrapper.Exit(result)
}

func TestWrappedChildHelperProcess(t *testing.T) {
	if os.Getenv("AAV_WRAPPED_CHILD_HELPER") != "1" {
		return
	}
	input, _ := io.ReadAll(os.Stdin)
	_, _ = fmt.Fprintf(os.Stdout, "stdout:%s\n", input)
	_, _ = fmt.Fprintf(os.Stderr, "stderr:%s\n", input)
	os.Exit(23)
}
