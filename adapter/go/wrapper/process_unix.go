//go:build !windows

package wrapper

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

func configureProcess(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func relaySignals(process *os.Process) func() {
	signals := make(chan os.Signal, 4)
	done := make(chan struct{})
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
	go func() {
		for {
			select {
			case value := <-signals:
				if native, ok := value.(syscall.Signal); ok {
					_ = syscall.Kill(-process.Pid, native)
				}
			case <-done:
				return
			}
		}
	}()
	return func() {
		signal.Stop(signals)
		close(done)
	}
}

func interruptProcess(process *os.Process) {
	_ = syscall.Kill(-process.Pid, syscall.SIGTERM)
}

func killProcess(process *os.Process) {
	_ = syscall.Kill(-process.Pid, syscall.SIGKILL)
}

func processResult(err error, state *os.ProcessState) Result {
	result := Result{}
	if err == nil {
		return result
	}
	if state == nil {
		result.ExitCode = 127
		result.StartError = err
		return result
	}
	status, _ := state.Sys().(syscall.WaitStatus)
	if status.Signaled() {
		value := status.Signal()
		result.Signal = value.String()
		result.processSignal = value
		result.ExitCode = 128 + int(value)
		return result
	}
	result.ExitCode = state.ExitCode()
	return result
}

// Exit terminates the wrapper with the same signal or exit code as the child.
func Exit(result Result) {
	if result.processSignal != nil {
		signal.Reset(result.processSignal)
		if native, ok := result.processSignal.(syscall.Signal); ok {
			_ = syscall.Kill(os.Getpid(), native)
			time.Sleep(50 * time.Millisecond)
		}
	}
	os.Exit(result.ExitCode)
}
