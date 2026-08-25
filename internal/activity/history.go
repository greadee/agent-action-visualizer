package activity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	pathpkg "path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/greadee/agent-action-visualizer/internal/project"
)

const reportPath = ".aav/activity-v1.json"

type Report struct {
	SchemaVersion int                   `json:"schema_version"`
	Files         map[string]FileReport `json:"files"`
}

type FileReport struct {
	TotalTimeMS    int64    `json:"total_time_ms"`
	LinesAdded     int64    `json:"lines_added"`
	LinesDeleted   int64    `json:"lines_deleted"`
	RecentTools    []string `json:"recent_tools"`
	SessionHistory []string `json:"session_history"`
}

// Enrich combines inferred Git history with explicitly agent-reported work metrics.
func Enrich(ctx context.Context, root string, nodes []project.Node) ([]project.Node, error) {
	history, err := gitHistory(ctx, root)
	if err != nil && !errors.Is(err, errNotRepository) {
		return nil, err
	}
	report, err := readReport(root)
	if err != nil {
		return nil, err
	}
	out := append([]project.Node(nil), nodes...)
	index := make(map[string]int, len(out))
	for i := range out {
		index[out[i].Path] = i
		activityByPath := history[out[i].Path]
		if reported, ok := report.Files[out[i].Path]; ok {
			activityByPath.TotalTimeMS = reported.TotalTimeMS
			activityByPath.LinesAdded = reported.LinesAdded
			activityByPath.LinesDeleted = reported.LinesDeleted
			activityByPath.RecentTools = mergeTools(reported.RecentTools, activityByPath.RecentTools)
			activityByPath.SessionHistory = append([]string(nil), reported.SessionHistory...)
			activityByPath.Confidence = "reported+git"
		} else if activityByPath.LastCommit != "" {
			activityByPath.Confidence = "git-inferred"
		}
		out[i].Activity = activityByPath
	}
	aggregateDirectories(out, index)
	return out, nil
}

var errNotRepository = errors.New("not a git repository")

func gitHistory(ctx context.Context, root string) (map[string]project.NodeActivity, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", root, "log", "--max-count=5000", "--no-renames", "--format=%x1e%H%x1f%ct%x1f%s", "--name-only")
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && bytes.Contains(exitErr.Stderr, []byte("not a git repository")) {
			return map[string]project.NodeActivity{}, errNotRepository
		}
		return nil, fmt.Errorf("read git history: %w", err)
	}
	result := map[string]project.NodeActivity{}
	for _, record := range bytes.Split(output, []byte{0x1e}) {
		fields := bytes.SplitN(record, []byte{0x1f}, 3)
		if len(fields) != 3 {
			continue
		}
		commit := strings.TrimSpace(string(fields[0]))
		stamp, _ := strconv.ParseInt(strings.TrimSpace(string(fields[1])), 10, 64)
		lines := strings.Split(strings.TrimSpace(string(fields[2])), "\n")
		if len(lines) == 0 {
			continue
		}
		subject := strings.TrimSpace(lines[0])
		tools := inferTools(subject)
		seen := map[string]bool{}
		for _, rawPath := range lines[1:] {
			path := filepath.ToSlash(strings.TrimSpace(rawPath))
			if path == "" || seen[path] {
				continue
			}
			seen[path] = true
			item := result[path]
			item.AccessCount++
			item.RecentTools = mergeTools(item.RecentTools, tools)
			if item.LastCommit == "" {
				item.GroupKey = commit
				item.LastCommit = commit
				item.LastCommitAt = stamp
				item.LastEvent = "commit " + shortCommit(commit) + ": " + subject
			}
			result[path] = item
		}
	}
	return result, nil
}

func readReport(root string) (Report, error) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(reportPath)))
	if errors.Is(err, os.ErrNotExist) {
		return Report{SchemaVersion: 1, Files: map[string]FileReport{}}, nil
	}
	if err != nil {
		return Report{}, fmt.Errorf("read agent activity report: %w", err)
	}
	var report Report
	if err := json.Unmarshal(data, &report); err != nil {
		return Report{}, fmt.Errorf("decode agent activity report: %w", err)
	}
	if report.SchemaVersion != 1 {
		return Report{}, fmt.Errorf("unsupported activity report schema_version %d", report.SchemaVersion)
	}
	normalized := make(map[string]FileReport, len(report.Files))
	for rawPath, item := range report.Files {
		cleaned := pathpkg.Clean(strings.ReplaceAll(rawPath, "\\", "/"))
		if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") || pathpkg.IsAbs(cleaned) {
			return Report{}, fmt.Errorf("activity report path must be project-relative: %q", rawPath)
		}
		if item.TotalTimeMS < 0 || item.LinesAdded < 0 || item.LinesDeleted < 0 {
			return Report{}, fmt.Errorf("activity report metrics must be non-negative for %q", rawPath)
		}
		normalized[cleaned] = item
	}
	report.Files = normalized
	return report, nil
}

func aggregateDirectories(nodes []project.Node, index map[string]int) {
	ordered := make([]int, len(nodes))
	for i := range nodes {
		ordered[i] = i
	}
	sort.Slice(ordered, func(i, j int) bool {
		return strings.Count(nodes[ordered[i]].Path, "/") > strings.Count(nodes[ordered[j]].Path, "/")
	})
	for _, nodeIndex := range ordered {
		node := nodes[nodeIndex]
		if node.Path == "." || node.Activity.LastCommit == "" {
			continue
		}
		parentIndex, ok := index[node.ParentPath]
		if !ok {
			continue
		}
		parent := &nodes[parentIndex]
		parent.Activity.AccessCount += node.Activity.AccessCount
		parent.Activity.TotalTimeMS += node.Activity.TotalTimeMS
		parent.Activity.LinesAdded += node.Activity.LinesAdded
		parent.Activity.LinesDeleted += node.Activity.LinesDeleted
		parent.Activity.RecentTools = mergeTools(parent.Activity.RecentTools, node.Activity.RecentTools)
		parent.Activity.SessionHistory = mergeTools(parent.Activity.SessionHistory, node.Activity.SessionHistory)
		if node.Activity.LastCommitAt > parent.Activity.LastCommitAt {
			parent.Activity.GroupKey = node.Activity.GroupKey
			parent.Activity.LastCommit = node.Activity.LastCommit
			parent.Activity.LastCommitAt = node.Activity.LastCommitAt
			parent.Activity.LastEvent = node.Activity.LastEvent
			parent.Activity.Confidence = node.Activity.Confidence
		}
	}
}

func inferTools(subject string) []string {
	tools := []string{"git commit"}
	lower := strings.ToLower(subject)
	if start := strings.Index(lower, "[tool:"); start >= 0 {
		if end := strings.Index(lower[start:], "]"); end > 6 {
			tools = append([]string{subject[start+6 : start+end]}, tools...)
		}
	}
	return tools
}

func mergeTools(groups ...[]string) []string {
	seen := map[string]bool{}
	var result []string
	for _, group := range groups {
		for _, tool := range group {
			tool = strings.TrimSpace(tool)
			if tool != "" && !seen[tool] {
				seen[tool] = true
				result = append(result, tool)
				if len(result) == 6 {
					return result
				}
			}
		}
	}
	return result
}

func shortCommit(commit string) string {
	if len(commit) > 8 {
		return commit[:8]
	}
	return commit
}
