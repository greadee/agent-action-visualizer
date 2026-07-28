package install

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

const (
	installID    = "aav-codex-hooks-v1"
	statusLabel  = "Agent Action Visualizer"
	hookTimeout  = 1
	maxConfigLen = 4 << 20
)

var hookEvents = []string{
	"SessionStart",
	"SessionEnd",
	"SubagentStart",
	"SubagentStop",
	"PreToolUse",
	"PostToolUse",
}

type hookHandler struct {
	Type           string `json:"type"`
	Command        string `json:"command"`
	CommandWindows string `json:"commandWindows"`
	Timeout        int    `json:"timeout"`
	StatusMessage  string `json:"statusMessage"`
}

type hookGroup struct {
	Matcher string          `json:"matcher,omitempty"`
	Hooks   json.RawMessage `json:"hooks"`
}

func mergeHooks(payload []byte, binaryPath string) ([]byte, int, error) {
	root, err := decodeRoot(payload)
	if err != nil {
		return nil, 0, err
	}
	hooks, err := decodeHooks(root["hooks"])
	if err != nil {
		return nil, 0, err
	}
	removed := removeManagedHooks(hooks)
	handler, err := managedHandler(binaryPath)
	if err != nil {
		return nil, 0, err
	}
	encodedHandler, err := json.Marshal(handler)
	if err != nil {
		return nil, 0, err
	}
	encodedGroup, err := json.Marshal(hookGroup{
		Hooks: json.RawMessage("[" + string(encodedHandler) + "]"),
	})
	if err != nil {
		return nil, 0, err
	}
	for _, event := range hookEvents {
		hooks[event] = append(hooks[event], encodedGroup)
	}
	encodedHooks, err := json.Marshal(hooks)
	if err != nil {
		return nil, 0, err
	}
	root["hooks"] = encodedHooks
	merged, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, 0, err
	}
	return append(merged, '\n'), removed, nil
}

func removeHooks(payload []byte) ([]byte, int, error) {
	root, err := decodeRoot(payload)
	if err != nil {
		return nil, 0, err
	}
	hooks, err := decodeHooks(root["hooks"])
	if err != nil {
		return nil, 0, err
	}
	_, hooksExisted := root["hooks"]
	removed := removeManagedHooks(hooks)
	if len(hooks) == 0 {
		if hooksExisted {
			root["hooks"] = json.RawMessage(`{}`)
		} else {
			delete(root, "hooks")
		}
	} else {
		encodedHooks, marshalErr := json.Marshal(hooks)
		if marshalErr != nil {
			return nil, 0, marshalErr
		}
		root["hooks"] = encodedHooks
	}
	if len(root) == 0 {
		return nil, removed, nil
	}
	cleaned, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, 0, err
	}
	return append(cleaned, '\n'), removed, nil
}

func onlyEmptyHooks(payload []byte) bool {
	root, err := decodeRoot(payload)
	if err != nil || len(root) != 1 {
		return false
	}
	hooks, err := decodeHooks(root["hooks"])
	return err == nil && len(hooks) == 0
}

func managedHookCount(payload []byte) (int, error) {
	count, _, err := managedHookLayout(payload)
	return count, err
}

func managedHookLayout(payload []byte) (int, bool, error) {
	root, err := decodeRoot(payload)
	if err != nil {
		return 0, false, err
	}
	hooks, err := decodeHooks(root["hooks"])
	if err != nil {
		return 0, false, err
	}
	count := 0
	perEvent := make(map[string]int, len(hookEvents))
	for event, groups := range hooks {
		for _, group := range groups {
			_, managed, _ := stripManagedHooks(group)
			count += managed
			perEvent[event] += managed
		}
	}
	complete := count == len(hookEvents)
	for _, event := range hookEvents {
		complete = complete && perEvent[event] == 1
	}
	return count, complete, nil
}

func decodeRoot(payload []byte) (map[string]json.RawMessage, error) {
	if len(payload) == 0 {
		return map[string]json.RawMessage{}, nil
	}
	if len(payload) > maxConfigLen {
		return nil, fmt.Errorf("hooks configuration exceeds %d bytes", maxConfigLen)
	}
	var root map[string]json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(payload))
	if err := decoder.Decode(&root); err != nil {
		return nil, fmt.Errorf("parse hooks configuration: %w", err)
	}
	if root == nil {
		return nil, errors.New("hooks configuration must be a JSON object")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, errors.New("hooks configuration contains trailing JSON")
	}
	return root, nil
}

func decodeHooks(payload json.RawMessage) (map[string][]json.RawMessage, error) {
	if len(payload) == 0 || bytes.Equal(bytes.TrimSpace(payload), []byte("null")) {
		return map[string][]json.RawMessage{}, nil
	}
	var hooks map[string][]json.RawMessage
	if err := json.Unmarshal(payload, &hooks); err != nil {
		return nil, fmt.Errorf("parse hooks object: %w", err)
	}
	if hooks == nil {
		return nil, errors.New("hooks must be a JSON object")
	}
	return hooks, nil
}

func removeManagedHooks(hooks map[string][]json.RawMessage) int {
	removed := 0
	for event, groups := range hooks {
		kept := groups[:0]
		for _, group := range groups {
			cleaned, count, keep := stripManagedHooks(group)
			removed += count
			if keep {
				kept = append(kept, cleaned)
			}
		}
		if len(kept) == 0 {
			delete(hooks, event)
		} else {
			hooks[event] = kept
		}
	}
	return removed
}

func stripManagedHooks(payload json.RawMessage) (json.RawMessage, int, bool) {
	var group map[string]json.RawMessage
	if json.Unmarshal(payload, &group) != nil || group == nil {
		return payload, 0, true
	}
	var handlers []json.RawMessage
	if json.Unmarshal(group["hooks"], &handlers) != nil {
		return payload, 0, true
	}
	kept := handlers[:0]
	removed := 0
	for _, payload := range handlers {
		var handler struct {
			Command       string `json:"command"`
			StatusMessage string `json:"statusMessage"`
		}
		if json.Unmarshal(payload, &handler) == nil &&
			handler.StatusMessage == statusLabel &&
			strings.Contains(handler.Command, installID) {
			removed++
			continue
		}
		kept = append(kept, payload)
	}
	if removed == 0 {
		return payload, 0, true
	}
	if len(kept) == 0 {
		return nil, removed, false
	}
	encodedHooks, err := json.Marshal(kept)
	if err != nil {
		return payload, 0, true
	}
	group["hooks"] = encodedHooks
	cleaned, err := json.Marshal(group)
	if err != nil {
		return payload, 0, true
	}
	return cleaned, removed, true
}

func managedHandler(binaryPath string) (hookHandler, error) {
	absolute, err := filepath.Abs(binaryPath)
	if err != nil {
		return hookHandler{}, fmt.Errorf("resolve hook binary: %w", err)
	}
	if strings.ContainsRune(absolute, 0) || strings.ContainsAny(absolute, "\r\n") {
		return hookHandler{}, errors.New("hook binary path contains unsupported control characters")
	}
	windows, err := quoteWindows(absolute)
	if err != nil {
		return hookHandler{}, err
	}
	return hookHandler{
		Type:           "command",
		Command:        quotePOSIX(absolute) + " --aav-install-id " + quotePOSIX(installID),
		CommandWindows: windows + ` --aav-install-id "` + installID + `"`,
		Timeout:        hookTimeout,
		StatusMessage:  statusLabel,
	}, nil
}

func quotePOSIX(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func quoteWindows(value string) (string, error) {
	if strings.Contains(value, `"`) {
		return "", errors.New("hook binary path contains an unsupported quote")
	}
	return `"` + value + `"`, nil
}

func semanticEqual(left, right []byte) bool {
	var leftValue any
	var rightValue any
	if json.Unmarshal(left, &leftValue) != nil || json.Unmarshal(right, &rightValue) != nil {
		return false
	}
	return reflectJSON(leftValue, rightValue)
}

func reflectJSON(left, right any) bool {
	leftBytes, leftErr := json.Marshal(left)
	rightBytes, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftBytes, rightBytes)
}
