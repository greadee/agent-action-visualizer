package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	codexinstall "github.com/greadee/agent-action-visualizer/internal/adapters/codex/install"
	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}
	switch args[0] {
	case "mock-event":
		return runMockEvent(args[1:], stdin, stdout, stderr)
	case "codex":
		return runCodex(ctx, args[1:], stdout, stderr)
	default:
		printUsage(stderr)
		return 2
	}
}

func runMockEvent(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) > 1 {
		fmt.Fprintln(stderr, "usage: aav mock-event [event.json|-]")
		return 2
	}
	reader := stdin
	var file *os.File
	if len(args) == 1 && args[0] != "-" {
		var err error
		file, err = os.Open(args[0])
		if err != nil {
			fmt.Fprintln(stderr, "open event file:", err)
			return 2
		}
		defer file.Close()
		reader = file
	}
	decoder := json.NewDecoder(io.LimitReader(reader, 1<<20))
	var event protocol.Event
	if err := decoder.Decode(&event); err != nil {
		fmt.Fprintln(stderr, "decode event:", err)
		return 2
	}
	if err := event.Validate(); err != nil {
		fmt.Fprintln(stderr, "validate event:", err)
		return 2
	}
	payload, err := json.MarshalIndent(event, "", "  ")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	fmt.Fprintln(stdout, string(payload))
	return 0
}

func runCodex(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printCodexUsage(stderr)
		return 2
	}
	action := args[0]
	if action != "install" && action != "uninstall" && action != "status" && action != "test" {
		printCodexUsage(stderr)
		return 2
	}
	flags := flag.NewFlagSet("aav codex "+action, flag.ContinueOnError)
	flags.SetOutput(stderr)
	scope := flags.String("scope", string(codexinstall.ScopeProject), "installation scope: project or user")
	project := flags.String("project", "", "project root (defaults to current directory)")
	codexHome := flags.String("codex-home", "", "Codex home for user scope")
	hookBinary := flags.String("hook-binary", "", "path to aav-codex-hook executable")
	dryRun := flags.Bool("dry-run", false, "report changes without writing files")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "unexpected positional arguments")
		return 2
	}
	if (action == "status" || action == "test") && *dryRun {
		fmt.Fprintln(stderr, "--dry-run is supported only by install and uninstall")
		return 2
	}
	options := codexinstall.Options{
		Scope:       codexinstall.Scope(strings.ToLower(strings.TrimSpace(*scope))),
		ProjectRoot: *project, CodexHome: *codexHome, HookBinary: *hookBinary, DryRun: *dryRun,
	}
	if action == "install" && options.HookBinary == "" {
		options.HookBinary = adjacentHookBinary()
	}
	manager := codexinstall.NewManager()
	var result codexinstall.Result
	var err error
	switch action {
	case "install":
		result, err = manager.Install(ctx, options)
	case "uninstall":
		result, err = manager.Uninstall(ctx, options)
	case "status":
		result, err = manager.Status(ctx, options)
	case "test":
		result, err = manager.Test(ctx, options)
	}
	if err != nil {
		fmt.Fprintf(stderr, "codex %s failed: %s\n", action, sanitizeDiagnostic(err))
		return 1
	}
	printResult(stdout, result)
	return 0
}

func adjacentHookBinary() string {
	executable, err := os.Executable()
	if err != nil {
		return "aav-codex-hook" + executableSuffix()
	}
	return filepath.Join(filepath.Dir(executable), "aav-codex-hook"+executableSuffix())
}

func executableSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

func sanitizeDiagnostic(err error) string {
	if err == nil {
		return ""
	}
	message := strings.ReplaceAll(err.Error(), "\r", " ")
	message = strings.ReplaceAll(message, "\n", " ")
	if len(message) > 500 {
		message = message[:500] + "..."
	}
	return message
}

func printResult(writer io.Writer, result codexinstall.Result) {
	fmt.Fprintf(writer, "action: %s\n", result.Action)
	fmt.Fprintf(writer, "scope: %s\n", result.Scope)
	fmt.Fprintf(writer, "status: %s\n", result.Status)
	fmt.Fprintf(writer, "configuration: %s\n", result.ConfigPath)
	fmt.Fprintf(writer, "hook binary: %s\n", result.BinaryPath)
	fmt.Fprintf(writer, "managed hooks: %d\n", result.ManagedHooks)
	fmt.Fprintf(writer, "changed: %t\n", result.Changed)
	for _, detail := range result.Details {
		fmt.Fprintf(writer, "detail: %s\n", detail)
	}
}

func printUsage(writer io.Writer) {
	fmt.Fprintln(writer, "usage: aav <mock-event|codex> [arguments]")
}

func printCodexUsage(writer io.Writer) {
	fmt.Fprintln(writer, "usage: aav codex <install|uninstall|status|test> [flags]")
}
