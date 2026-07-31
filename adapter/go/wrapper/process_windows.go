//go:build windows

package wrapper

import (
	"os"
	"os/exec"
)

func configureProcess(*exec.Cmd) {}

// Windows console control events are delivered to the wrapper and child by
// the shared console. Context cancellation uses a process kill as a fallback.
func relaySignals(*os.Process) func() {
	return func() {}
}

func interruptProcess(process *os.Process) {
	_ = process.Kill()
}

func killProcess(process *os.Process) {
	_ = process.Kill()
}

func processResult(err error, state *os.ProcessState) Result {
	if err == nil {
		return Result{}
	}
	if state == nil {
		return Result{ExitCode: 127, StartError: err}
	}
	return Result{ExitCode: state.ExitCode()}
}

// Exit terminates the wrapper with the child's native Windows exit code.
func Exit(result Result) {
	os.Exit(result.ExitCode)
}
