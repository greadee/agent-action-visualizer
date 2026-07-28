package codex

import (
	"bufio"
	"path/filepath"
	"strings"

	protocol "github.com/greadee/agent-action-visualizer/protocol/go"
)

func parsePatch(patch string) []fileEvidence {
	if len(patch) == 0 || len(patch) > MaxHookInputBytes {
		return nil
	}
	var result []fileEvidence
	var current *fileEvidence
	finish := func() {
		if current == nil || safePath(current.path) == "" || len(result) >= MaxEventsPerHook-1 {
			current = nil
			return
		}
		result = append(result, *current)
		current = nil
	}
	scanner := bufio.NewScanner(strings.NewReader(patch))
	scanner.Buffer(make([]byte, 1024), MaxHookInputBytes)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "*** Add File: "):
			finish()
			zero := int64(0)
			current = &fileEvidence{eventType: protocol.EventFileCreated, operation: "create", path: strings.TrimPrefix(line, "*** Add File: "), linesAdded: &zero, linesDeleted: int64Pointer(0)}
		case strings.HasPrefix(line, "*** Update File: "):
			finish()
			current = &fileEvidence{eventType: protocol.EventFilePatched, operation: "patch", path: strings.TrimPrefix(line, "*** Update File: "), linesAdded: int64Pointer(0), linesDeleted: int64Pointer(0)}
		case strings.HasPrefix(line, "*** Delete File: "):
			finish()
			current = &fileEvidence{eventType: protocol.EventFileDeleted, operation: "delete", path: strings.TrimPrefix(line, "*** Delete File: "), linesAdded: int64Pointer(0), linesDeleted: int64Pointer(0)}
		case strings.HasPrefix(line, "*** Move to: ") && current != nil:
			oldPath := current.path
			newPath := strings.TrimPrefix(line, "*** Move to: ")
			current.eventType, current.operation = moveKind(oldPath, newPath)
			current.previousPath, current.path = oldPath, newPath
		case current != nil && strings.HasPrefix(line, "+"):
			*current.linesAdded++
		case current != nil && strings.HasPrefix(line, "-"):
			*current.linesDeleted++
		}
	}
	finish()
	if scanner.Err() != nil {
		return nil
	}
	return result
}

func moveKind(oldPath, newPath string) (protocol.EventType, string) {
	oldDir := filepath.ToSlash(filepath.Dir(oldPath))
	newDir := filepath.ToSlash(filepath.Dir(newPath))
	if oldDir == newDir {
		return protocol.EventFileRenamed, "rename"
	}
	return protocol.EventFileMoved, "move"
}

func int64Pointer(value int64) *int64 { return &value }
