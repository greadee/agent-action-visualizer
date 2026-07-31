package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/greadee/agent-action-visualizer/adapter/go/wrapper"
	filesystemadapter "github.com/greadee/agent-action-visualizer/internal/adapters/filesystem"
	"github.com/greadee/agent-action-visualizer/internal/ipc"
)

func main() {
	result, usageCode := run(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	if usageCode != 0 {
		os.Exit(usageCode)
	}
	wrapper.Exit(result)
}

func run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) (wrapper.Result, int) {
	flags := flag.NewFlagSet("aav-wrapper", flag.ContinueOnError)
	flags.SetOutput(stderr)
	projectRoot := flags.String("project-root", "", "project root associated with the command")
	sessionID := flags.String("session-id", "", "stable session id (generated when omitted)")
	agentType := flags.String("agent-type", "generic", "agent type label")
	endpoint := flags.String("endpoint", ipc.DefaultEndpoint(), "local AAV collector endpoint")
	filesystemFallback := flags.Bool("filesystem-fallback", true, "observe repository changes when stronger evidence is unavailable")
	if err := flags.Parse(args); err != nil {
		return wrapper.Result{}, 2
	}
	command := flags.Args()
	if len(command) == 0 {
		fmt.Fprintln(stderr, "usage: aav-wrapper [flags] -- command [arguments...]")
		return wrapper.Result{}, 2
	}
	config := wrapper.Config{
		Command:     command[0],
		Args:        command[1:],
		ProjectRoot: *projectRoot,
		SessionID:   *sessionID,
		AgentType:   strings.TrimSpace(*agentType),
		Stdin:       stdin,
		Stdout:      stdout,
		Stderr:      stderr,
		Collector:   ipc.NewClient(*endpoint),
	}
	if *filesystemFallback {
		config.Fallback = filesystemadapter.NewObserver(filesystemadapter.Config{})
	}
	result := wrapper.Run(ctx, config)
	if result.StartError != nil {
		fmt.Fprintf(stderr, "aav-wrapper: %s\n", sanitize(result.StartError.Error()))
	}
	return result, 0
}

func sanitize(value string) string {
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	if len(value) > 500 {
		return value[:500] + "..."
	}
	return value
}
