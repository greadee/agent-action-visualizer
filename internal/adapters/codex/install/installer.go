// Package install manages reversible Codex hooks.json integration.
package install

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	stateVersion = 1
	maxStateLen  = 64 << 10
	maxBinaryLen = 128 << 20
)

type Scope string

const (
	ScopeProject Scope = "project"
	ScopeUser    Scope = "user"
)

type Options struct {
	Scope       Scope
	ProjectRoot string
	CodexHome   string
	HookBinary  string
	DryRun      bool
}

type Result struct {
	Action       string
	Scope        Scope
	Status       string
	ConfigPath   string
	BinaryPath   string
	Changed      bool
	ManagedHooks int
	Details      []string
}

type state struct {
	Version        int    `json:"version"`
	InstallID      string `json:"install_id"`
	Scope          Scope  `json:"scope"`
	Phase          string `json:"phase"`
	ConfigPath     string `json:"config_path"`
	BinaryPath     string `json:"binary_path"`
	OriginalExists bool   `json:"original_exists"`
	OriginalSHA256 string `json:"original_sha256,omitempty"`
	BackupPath     string `json:"backup_path,omitempty"`
	BinarySHA256   string `json:"binary_sha256"`
}

type target struct {
	scope      Scope
	configDir  string
	configPath string
	statePath  string
	backupPath string
	binaryPath string
}

type Manager struct {
	probe func(context.Context, string) error
}

func NewManager() *Manager {
	return &Manager{probe: probeInstalledHook}
}

func (m *Manager) Install(_ context.Context, options Options) (Result, error) {
	target, err := resolveTarget(options)
	if err != nil {
		return Result{}, err
	}
	result := baseResult("install", target)
	if options.DryRun {
		if err := checkInterruptedWrites(target); err != nil {
			return result, err
		}
	} else if err := recoverTarget(target); err != nil {
		return result, fmt.Errorf("recover interrupted write: %w", err)
	}
	source, err := filepath.Abs(options.HookBinary)
	if err != nil || strings.TrimSpace(options.HookBinary) == "" {
		return result, errors.New("hook binary is required")
	}
	binary, found, err := readOptional(source, maxBinaryLen)
	if err != nil {
		return result, fmt.Errorf("read hook binary: %w", err)
	}
	if !found {
		return result, errors.New("hook binary does not exist")
	}
	config, configExists, err := readOptional(target.configPath, maxConfigLen)
	if err != nil {
		return result, fmt.Errorf("read hooks configuration: %w", err)
	}
	if configExists && len(config) == 0 {
		return result, errors.New("hooks configuration is empty rather than valid JSON")
	}
	desired, _, err := mergeHooks(config, target.binaryPath)
	if err != nil {
		return result, err
	}
	previous, stateExists, err := readState(target.statePath)
	if err != nil {
		return result, err
	}
	if stateExists {
		if err := validateState(previous, target); err != nil {
			return result, err
		}
		if options.DryRun {
			if err := validateBackup(previous); err != nil {
				return result, err
			}
		} else if err := ensureBackup(previous, config, configExists); err != nil {
			return result, err
		}
	}
	installedBinary, installedExists, err := readOptional(target.binaryPath, maxBinaryLen)
	if err != nil {
		return result, fmt.Errorf("read installed hook: %w", err)
	}
	if !stateExists && installedExists {
		return result, errors.New("unmanaged file exists at the hook binary destination")
	}
	result.ManagedHooks = len(hookEvents)
	binarySHA := digest(binary)
	result.Changed = !configExists || !bytes.Equal(config, desired) || !installedExists || !bytes.Equal(installedBinary, binary) || !stateExists || previous.Phase != "installed" || previous.BinarySHA256 != binarySHA
	if !result.Changed {
		result.Status = "installed"
		result.Details = []string{"configuration and managed hook binary are current"}
		return result, nil
	}
	if options.DryRun {
		result.Status = "planned"
		result.Details = []string{"no files were changed"}
		return result, nil
	}
	currentState := previous
	if !stateExists {
		currentState = state{
			Version: stateVersion, InstallID: installID, Scope: target.scope, Phase: "installing",
			ConfigPath: target.configPath, BinaryPath: target.binaryPath, OriginalExists: configExists,
			BinarySHA256: binarySHA,
		}
		if configExists {
			_, backupExists, backupErr := readOptional(target.backupPath, maxConfigLen)
			if backupErr != nil {
				return result, backupErr
			} else if backupExists {
				return result, errors.New("unmanaged installer backup already exists")
			}
			currentState.BackupPath = target.backupPath
			currentState.OriginalSHA256 = digest(config)
		}
		if err := writeState(target.statePath, currentState); err != nil {
			return result, err
		}
		if err := ensureBackup(currentState, config, configExists); err != nil {
			return result, err
		}
	}
	if err := atomicWrite(target.binaryPath, binary, 0o700); err != nil {
		return result, fmt.Errorf("install hook binary: %w", err)
	}
	if err := atomicWrite(target.configPath, desired, 0o600); err != nil {
		return result, fmt.Errorf("write hooks configuration: %w", err)
	}
	currentState.Phase = "installed"
	currentState.BinarySHA256 = binarySHA
	if err := writeState(target.statePath, currentState); err != nil {
		return result, err
	}
	result.Status = "installed"
	result.Details = []string{"managed hook handlers installed", "Codex may require hook review before execution"}
	return result, nil
}

func (m *Manager) Uninstall(_ context.Context, options Options) (Result, error) {
	target, err := resolveTarget(options)
	if err != nil {
		return Result{}, err
	}
	result := baseResult("uninstall", target)
	if options.DryRun {
		if err := checkInterruptedWrites(target); err != nil {
			return result, err
		}
	} else if err := recoverTarget(target); err != nil {
		return result, fmt.Errorf("recover interrupted write: %w", err)
	}
	config, configExists, err := readOptional(target.configPath, maxConfigLen)
	if err != nil {
		return result, fmt.Errorf("read hooks configuration: %w", err)
	}
	if configExists && len(config) == 0 {
		return result, errors.New("hooks configuration is empty rather than valid JSON")
	}
	currentState, stateExists, err := readState(target.statePath)
	if err != nil {
		return result, err
	}
	if stateExists {
		if err := validateState(currentState, target); err != nil {
			return result, err
		}
		if options.DryRun {
			if err := validateBackup(currentState); err != nil {
				return result, err
			}
		} else if err := ensureBackup(currentState, config, configExists); err != nil {
			return result, err
		}
	}
	cleaned := config
	removed := 0
	if configExists {
		cleaned, removed, err = removeHooks(config)
		if err != nil {
			return result, err
		}
	}
	result.ManagedHooks = removed
	_, binaryExists, binaryErr := readOptional(target.binaryPath, maxBinaryLen)
	if binaryErr != nil {
		return result, binaryErr
	}
	if !stateExists && binaryExists {
		return result, errors.New("unmanaged file exists at the hook binary destination")
	}
	result.Changed = stateExists || binaryExists || removed > 0
	if !result.Changed {
		result.Status = "not-installed"
		result.Details = []string{"no managed installation was found"}
		return result, nil
	}
	restore, restoreExists, err := restoration(currentState, stateExists, cleaned)
	if err != nil {
		return result, err
	}
	if stateExists && !currentState.OriginalExists && onlyEmptyHooks(cleaned) {
		cleaned = nil
	}
	if options.DryRun {
		result.Status = "planned"
		result.Details = []string{"no files were changed"}
		return result, nil
	}
	if restoreExists {
		if err := atomicWrite(target.configPath, restore, 0o600); err != nil {
			return result, fmt.Errorf("restore hooks configuration: %w", err)
		}
	} else if len(cleaned) == 0 {
		if err := removeIfExists(target.configPath); err != nil {
			return result, fmt.Errorf("remove hooks configuration: %w", err)
		}
	} else if err := atomicWrite(target.configPath, cleaned, 0o600); err != nil {
		return result, fmt.Errorf("write hooks configuration: %w", err)
	}
	if err := removeIfExists(target.binaryPath); err != nil {
		return result, fmt.Errorf("remove hook binary: %w", err)
	}
	if stateExists && currentState.BackupPath != "" {
		if err := removeIfExists(currentState.BackupPath); err != nil {
			return result, fmt.Errorf("remove configuration backup: %w", err)
		}
	}
	if err := removeIfExists(target.statePath); err != nil {
		return result, fmt.Errorf("remove installer state: %w", err)
	}
	removeEmptyManagedDirs(target)
	result.Status = "not-installed"
	result.Details = []string{"managed handlers and binary removed", "unrelated hooks were preserved"}
	return result, nil
}

func (m *Manager) Status(_ context.Context, options Options) (Result, error) {
	target, err := resolveTarget(options)
	if err != nil {
		return Result{}, err
	}
	result := baseResult("status", target)
	if err := checkInterruptedWrites(target); err != nil {
		result.Status = "partial"
		result.Details = []string{"an interrupted write requires install or uninstall recovery"}
		return result, nil
	}
	config, configExists, err := readOptional(target.configPath, maxConfigLen)
	if err != nil {
		return result, err
	}
	if configExists && len(config) == 0 {
		result.Status = "invalid"
		return result, errors.New("hooks configuration is empty rather than valid JSON")
	}
	count := 0
	layoutComplete := false
	if configExists {
		count, layoutComplete, err = managedHookLayout(config)
		if err != nil {
			result.Status = "invalid"
			return result, err
		}
	}
	installation, stateExists, err := readState(target.statePath)
	if err != nil {
		result.Status = "invalid"
		return result, err
	}
	if stateExists {
		if err := validateState(installation, target); err != nil {
			result.Status = "invalid"
			return result, err
		}
		if err := validateBackup(installation); err != nil {
			result.Status = "invalid"
			return result, err
		}
	}
	binary, binaryExists, err := readOptional(target.binaryPath, maxBinaryLen)
	if err != nil {
		return result, err
	}
	result.ManagedHooks = count
	switch {
	case !stateExists && !binaryExists && count == 0:
		result.Status = "not-installed"
		result.Details = []string{"no managed installation was found"}
	case stateExists && installation.Phase == "installed" && binaryExists && digest(binary) == installation.BinarySHA256 && layoutComplete:
		result.Status = "installed"
		result.Details = []string{"managed configuration and hook binary are present"}
	default:
		result.Status = "partial"
		result.Details = []string{"installation is incomplete; rerun install or uninstall"}
	}
	return result, nil
}

func (m *Manager) Test(ctx context.Context, options Options) (Result, error) {
	result, err := m.Status(ctx, options)
	result.Action = "test"
	if err != nil {
		return result, err
	}
	if result.Status != "installed" {
		return result, errors.New("Codex hook is not fully installed")
	}
	if err := m.probe(ctx, result.BinaryPath); err != nil {
		result.Status = "failed"
		return result, err
	}
	result.Status = "passed"
	result.Details = []string{"installed hook emitted a sanitized event to an isolated local collector", "hook stdout and stderr were empty"}
	return result, nil
}

func resolveTarget(options Options) (target, error) {
	scope := options.Scope
	if scope == "" {
		scope = ScopeProject
	}
	var configDir string
	switch scope {
	case ScopeProject:
		root := options.ProjectRoot
		if root == "" {
			var err error
			root, err = os.Getwd()
			if err != nil {
				return target{}, err
			}
		}
		absolute, err := filepath.Abs(root)
		if err != nil {
			return target{}, err
		}
		configDir = filepath.Join(absolute, ".codex")
	case ScopeUser:
		configDir = options.CodexHome
		if configDir == "" {
			configDir = strings.TrimSpace(os.Getenv("CODEX_HOME"))
		}
		if configDir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return target{}, err
			}
			configDir = filepath.Join(home, ".codex")
		}
		absolute, err := filepath.Abs(configDir)
		if err != nil {
			return target{}, err
		}
		configDir = absolute
	default:
		return target{}, fmt.Errorf("unsupported scope %q", scope)
	}
	binaryName := "aav-codex-hook"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	return target{
		scope: scope, configDir: configDir,
		configPath: filepath.Join(configDir, "hooks.json"),
		statePath:  filepath.Join(configDir, ".aav-codex-install.json"),
		backupPath: filepath.Join(configDir, "hooks.json.aav-backup"),
		binaryPath: filepath.Join(configDir, "aav", "bin", binaryName),
	}, nil
}

func baseResult(action string, target target) Result {
	return Result{Action: action, Scope: target.scope, ConfigPath: target.configPath, BinaryPath: target.binaryPath}
}

func readState(path string) (state, bool, error) {
	payload, found, err := readOptional(path, maxStateLen)
	if err != nil || !found {
		return state{}, found, err
	}
	var value state
	if err := json.Unmarshal(payload, &value); err != nil {
		return state{}, true, fmt.Errorf("parse installer state: %w", err)
	}
	return value, true, nil
}

func writeState(path string, value state) error {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := atomicWrite(path, append(payload, '\n'), 0o600); err != nil {
		return fmt.Errorf("write installer state: %w", err)
	}
	return nil
}

func validateState(value state, target target) error {
	if value.Version != stateVersion || value.InstallID != installID {
		return errors.New("installer state belongs to an unsupported installation")
	}
	if value.Scope != target.scope || filepath.Clean(value.ConfigPath) != filepath.Clean(target.configPath) || filepath.Clean(value.BinaryPath) != filepath.Clean(target.binaryPath) {
		return errors.New("installer state target does not match requested scope")
	}
	if value.Phase != "installing" && value.Phase != "installed" {
		return errors.New("installer state has an invalid phase")
	}
	if len(value.BinarySHA256) != 64 {
		return errors.New("installer state has an invalid binary digest")
	}
	return nil
}

func validateBackup(value state) error {
	if !value.OriginalExists {
		return nil
	}
	backup, found, err := readOptional(value.BackupPath, maxConfigLen)
	if err != nil {
		return err
	}
	if !found || digest(backup) != value.OriginalSHA256 {
		return errors.New("configuration backup is missing or does not match installer state")
	}
	return nil
}

func ensureBackup(value state, config []byte, configExists bool) error {
	if !value.OriginalExists {
		return nil
	}
	backup, found, err := readOptional(value.BackupPath, maxConfigLen)
	if err != nil {
		return err
	}
	if found {
		if digest(backup) != value.OriginalSHA256 {
			return errors.New("configuration backup does not match installer state")
		}
		return nil
	}
	if value.Phase != "installing" || !configExists || digest(config) != value.OriginalSHA256 {
		return errors.New("configuration backup is missing and cannot be safely recovered")
	}
	count, err := managedHookCount(config)
	if err != nil || count != 0 {
		return errors.New("configuration backup is missing and cannot be safely recovered")
	}
	if err := atomicWrite(value.BackupPath, config, 0o600); err != nil {
		return fmt.Errorf("recover configuration backup: %w", err)
	}
	return nil
}

func restoration(value state, stateExists bool, cleaned []byte) ([]byte, bool, error) {
	if !stateExists || !value.OriginalExists {
		return nil, false, nil
	}
	backup, found, err := readOptional(value.BackupPath, maxConfigLen)
	if err != nil {
		return nil, false, err
	}
	if !found || digest(backup) != value.OriginalSHA256 {
		return nil, false, errors.New("configuration backup is missing or does not match installer state")
	}
	if len(cleaned) == 0 {
		return backup, true, nil
	}
	if semanticEqual(cleaned, backup) {
		return backup, true, nil
	}
	return nil, false, nil
}

func removeEmptyManagedDirs(target target) {
	_ = os.Remove(filepath.Dir(target.binaryPath))
	_ = os.Remove(filepath.Join(target.configDir, "aav"))
}

func recoverTarget(target target) error {
	for _, path := range []string{target.configPath, target.statePath, target.backupPath, target.binaryPath} {
		if err := recoverSwap(path); err != nil {
			return err
		}
	}
	return nil
}

func checkInterruptedWrites(target target) error {
	for _, path := range []string{target.configPath, target.statePath, target.backupPath, target.binaryPath} {
		if _, err := os.Stat(path + ".aav-swap"); err == nil {
			return errors.New("interrupted installer write requires recovery")
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}
