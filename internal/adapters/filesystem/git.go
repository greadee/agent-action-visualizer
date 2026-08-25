package filesystem

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type GitStatus struct {
	Index    byte
	Worktree byte
}

type GitDelta struct {
	Added   int64
	Deleted int64
	Binary  bool
}

type GitEvidence struct {
	Status  map[string]GitStatus
	Renames map[string]string
	Deltas  map[string]GitDelta
}

type GitInspector interface {
	Inspect(context.Context, string, []string) (GitEvidence, error)
}

type CommandGitInspector struct {
	timeout time.Duration
}

func NewCommandGitInspector(timeout time.Duration) *CommandGitInspector {
	if timeout <= 0 {
		timeout = 250 * time.Millisecond
	}
	return &CommandGitInspector{timeout: timeout}
}

func (g *CommandGitInspector) Inspect(ctx context.Context, root string, paths []string) (GitEvidence, error) {
	deadline, cancel := context.WithTimeout(ctx, g.timeout)
	defer cancel()
	relative := relativePaths(root, paths)
	evidence := GitEvidence{
		Status:  make(map[string]GitStatus),
		Renames: make(map[string]string),
		Deltas:  make(map[string]GitDelta),
	}

	statusArgs := []string{"-C", root, "status", "--porcelain=v1", "-z", "--untracked-files=all", "--renames", "--"}
	statusOutput, err := exec.CommandContext(deadline, "git", append(statusArgs, relative...)...).Output()
	if err != nil {
		return GitEvidence{}, err
	}
	parseStatus(statusOutput, evidence.Status, evidence.Renames)

	diffArgs := []string{"-C", root, "diff", "--no-ext-diff", "--no-renames", "--numstat", "-z", "--"}
	diffOutput, err := exec.CommandContext(deadline, "git", append(diffArgs, relative...)...).Output()
	if err != nil {
		return evidence, err
	}
	parseNumstat(diffOutput, evidence.Deltas)
	return evidence, nil
}

func relativePaths(root string, paths []string) []string {
	result := make([]string, 0, len(paths))
	seen := make(map[string]bool, len(paths))
	for _, path := range paths {
		relative, err := filepath.Rel(root, path)
		if err != nil || relative == "." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			continue
		}
		relative = filepath.ToSlash(relative)
		if !seen[relative] {
			seen[relative] = true
			result = append(result, relative)
		}
	}
	return result
}

func parseStatus(output []byte, status map[string]GitStatus, renames map[string]string) {
	records := bytes.Split(output, []byte{0})
	for index := 0; index < len(records); index++ {
		record := records[index]
		if len(record) < 4 || record[2] != ' ' {
			continue
		}
		path := filepath.ToSlash(string(record[3:]))
		value := GitStatus{Index: record[0], Worktree: record[1]}
		status[path] = value
		if (record[0] == 'R' || record[1] == 'R') && index+1 < len(records) {
			previous := filepath.ToSlash(string(records[index+1]))
			if previous != "" {
				renames[previous] = path
			}
			index++
		}
	}
}

func parseNumstat(output []byte, deltas map[string]GitDelta) {
	for _, record := range bytes.Split(output, []byte{0}) {
		fields := bytes.SplitN(record, []byte{'\t'}, 3)
		if len(fields) != 3 {
			continue
		}
		path := filepath.ToSlash(string(fields[2]))
		if path == "" {
			continue
		}
		if string(fields[0]) == "-" || string(fields[1]) == "-" {
			deltas[path] = GitDelta{Binary: true}
			continue
		}
		added, addErr := strconv.ParseInt(string(fields[0]), 10, 64)
		deleted, deleteErr := strconv.ParseInt(string(fields[1]), 10, 64)
		if addErr == nil && deleteErr == nil {
			deltas[path] = GitDelta{Added: added, Deleted: deleted}
		}
	}
}
