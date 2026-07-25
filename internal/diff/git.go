package diff

import (
	"bufio"
	"context"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type CommandGitRunner struct {
	timeout time.Duration
}

func NewCommandGitRunner(timeout time.Duration) *CommandGitRunner {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	return &CommandGitRunner{timeout: timeout}
}

func (r *CommandGitRunner) Numstat(ctx context.Context, root string, paths []string) (map[string]GitDelta, error) {
	deadline, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	args := []string{"-C", root, "diff", "--no-ext-diff", "--numstat", "--"}
	args = append(args, paths...)
	output, err := exec.CommandContext(deadline, "git", args...).Output()
	if err != nil {
		return nil, err
	}
	result := make(map[string]GitDelta)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), "\t", 3)
		if len(parts) != 3 {
			continue
		}
		path := strings.Trim(parts[2], "\"")
		if parts[0] == "-" || parts[1] == "-" {
			result[path] = GitDelta{Binary: true}
			continue
		}
		added, addErr := strconv.ParseInt(parts[0], 10, 64)
		deleted, deleteErr := strconv.ParseInt(parts[1], 10, 64)
		if addErr == nil && deleteErr == nil {
			result[path] = GitDelta{Added: added, Deleted: deleted}
		}
	}
	return result, scanner.Err()
}
