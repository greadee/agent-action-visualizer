package install

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectInstallIsIdempotentAndRestoresExactConfiguration(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, ".codex")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	original := []byte("{\r\n  \"description\": \"exact formatting\",\r\n  \"hooks\": {}\r\n}\r\n")
	configPath := filepath.Join(configDir, "hooks.json")
	if err := os.WriteFile(configPath, original, 0o600); err != nil {
		t.Fatal(err)
	}
	source := writeFakeHook(t)
	manager := NewManager()
	options := Options{Scope: ScopeProject, ProjectRoot: root, HookBinary: source}

	first, err := manager.Install(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Changed || first.Status != "installed" || first.ManagedHooks != len(hookEvents) {
		t.Fatalf("first result = %#v", first)
	}
	installedConfig := mustRead(t, first.ConfigPath)
	stateInfo := mustStat(t, filepath.Join(configDir, ".aav-codex-install.json"))
	configInfo := mustStat(t, first.ConfigPath)
	binaryInfo := mustStat(t, first.BinaryPath)

	second, err := manager.Install(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	if second.Changed || second.Status != "installed" {
		t.Fatalf("second result = %#v", second)
	}
	if !bytes.Equal(installedConfig, mustRead(t, first.ConfigPath)) ||
		!stateInfo.ModTime().Equal(mustStat(t, filepath.Join(configDir, ".aav-codex-install.json")).ModTime()) ||
		!configInfo.ModTime().Equal(mustStat(t, first.ConfigPath).ModTime()) ||
		!binaryInfo.ModTime().Equal(mustStat(t, first.BinaryPath).ModTime()) {
		t.Fatal("idempotent install rewrote managed files")
	}

	status, err := manager.Status(context.Background(), options)
	if err != nil || status.Status != "installed" {
		t.Fatalf("status = %#v, err = %v", status, err)
	}
	uninstalled, err := manager.Uninstall(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	if uninstalled.Status != "not-installed" || !bytes.Equal(original, mustRead(t, configPath)) {
		t.Fatalf("uninstall = %#v, config = %q", uninstalled, mustRead(t, configPath))
	}
	assertMissing(t, first.BinaryPath)
	assertMissing(t, filepath.Join(configDir, ".aav-codex-install.json"))
	assertMissing(t, filepath.Join(configDir, "hooks.json.aav-backup"))
}

func TestUninstallPreservesUnrelatedChangesAfterInstall(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, ".codex")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "hooks.json")
	if err := os.WriteFile(configPath, []byte(`{"description":"before"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	manager := NewManager()
	options := Options{ProjectRoot: root, HookBinary: writeFakeHook(t)}
	installed, err := manager.Install(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	var rootConfig map[string]json.RawMessage
	if err := json.Unmarshal(mustRead(t, configPath), &rootConfig); err != nil {
		t.Fatal(err)
	}
	rootConfig["unrelated_after_install"] = json.RawMessage(`{"secret":"untouched"}`)
	modified, _ := json.Marshal(rootConfig)
	if err := os.WriteFile(configPath, modified, 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := manager.Uninstall(context.Background(), options); err != nil {
		t.Fatal(err)
	}
	cleaned := string(mustRead(t, configPath))
	if strings.Contains(cleaned, installID) || !strings.Contains(cleaned, `"unrelated_after_install"`) || !strings.Contains(cleaned, `"secret"`) {
		t.Fatalf("unrelated change was not preserved: %s", cleaned)
	}
	assertMissing(t, installed.BinaryPath)
}

func TestInstallAndUninstallWithoutOriginalConfiguration(t *testing.T) {
	root := t.TempDir()
	manager := NewManager()
	options := Options{ProjectRoot: root, HookBinary: writeFakeHook(t)}
	result, err := manager.Install(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(result.ConfigPath); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Uninstall(context.Background(), options); err != nil {
		t.Fatal(err)
	}
	assertMissing(t, result.ConfigPath)
}

func TestDryRunDoesNotCreateProjectOrUserFiles(t *testing.T) {
	for _, scope := range []Scope{ScopeProject, ScopeUser} {
		t.Run(string(scope), func(t *testing.T) {
			root := t.TempDir()
			options := Options{Scope: scope, ProjectRoot: root, CodexHome: root, HookBinary: writeFakeHook(t), DryRun: true}
			result, err := NewManager().Install(context.Background(), options)
			if err != nil {
				t.Fatal(err)
			}
			if result.Status != "planned" || !result.Changed {
				t.Fatalf("result = %#v", result)
			}
			assertMissing(t, result.ConfigPath)
			assertMissing(t, result.BinaryPath)
		})
	}
}

func TestUserInstallUsesExplicitCodexHome(t *testing.T) {
	codexHome := filepath.Join(t.TempDir(), "user codex home")
	manager := NewManager()
	options := Options{Scope: ScopeUser, CodexHome: codexHome, HookBinary: writeFakeHook(t)}
	result, err := manager.Install(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	if result.Scope != ScopeUser || result.ConfigPath != filepath.Join(codexHome, "hooks.json") {
		t.Fatalf("result = %#v", result)
	}
	if _, err := manager.Uninstall(context.Background(), options); err != nil {
		t.Fatal(err)
	}
	assertMissing(t, result.ConfigPath)
}

func TestInvalidConfigurationCausesNoWrites(t *testing.T) {
	for name, invalid := range map[string][]byte{"malformed": []byte(`{"hooks":`), "empty": {}} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			configDir := filepath.Join(root, ".codex")
			if err := os.MkdirAll(configDir, 0o700); err != nil {
				t.Fatal(err)
			}
			configPath := filepath.Join(configDir, "hooks.json")
			if err := os.WriteFile(configPath, invalid, 0o600); err != nil {
				t.Fatal(err)
			}
			result, err := NewManager().Install(context.Background(), Options{ProjectRoot: root, HookBinary: writeFakeHook(t)})
			if err == nil {
				t.Fatal("expected invalid JSON error")
			}
			if !bytes.Equal(invalid, mustRead(t, configPath)) {
				t.Fatal("invalid configuration was changed")
			}
			assertMissing(t, result.BinaryPath)
			assertMissing(t, filepath.Join(configDir, ".aav-codex-install.json"))
			assertMissing(t, filepath.Join(configDir, "hooks.json.aav-backup"))
		})
	}
}

func TestStatusReportsPartialAndTestUsesProbe(t *testing.T) {
	root := t.TempDir()
	manager := NewManager()
	options := Options{ProjectRoot: root, HookBinary: writeFakeHook(t)}
	installed, err := manager.Install(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(installed.BinaryPath, []byte("tampered hook"), 0o700); err != nil {
		t.Fatal(err)
	}
	status, err := manager.Status(context.Background(), options)
	if err != nil || status.Status != "partial" {
		t.Fatalf("status = %#v, err = %v", status, err)
	}
	if _, err := manager.Test(context.Background(), options); err == nil {
		t.Fatal("expected partial installation test error")
	}

	if _, err := manager.Install(context.Background(), options); err != nil {
		t.Fatal(err)
	}
	called := false
	manager.probe = func(_ context.Context, binary string) error {
		called = binary == installed.BinaryPath
		return nil
	}
	result, err := manager.Test(context.Background(), options)
	if err != nil || !called || result.Status != "passed" {
		t.Fatalf("test = %#v, called = %t, err = %v", result, called, err)
	}
	manager.probe = func(context.Context, string) error { return errors.New("probe failed") }
	if result, err := manager.Test(context.Background(), options); err == nil || result.Status != "failed" {
		t.Fatalf("failed probe result = %#v, err = %v", result, err)
	}
}

func TestAtomicWriteRecoversInterruptedSwap(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "hooks.json")
	if err := os.WriteFile(path+".aav-swap", []byte("before"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := atomicWrite(path, []byte("after"), 0o600); err != nil {
		t.Fatal(err)
	}
	if string(mustRead(t, path)) != "after" {
		t.Fatal("atomic write did not complete after swap recovery")
	}
	assertMissing(t, path+".aav-swap")
}

func TestDryRunAndStatusDoNotRecoverInterruptedSwap(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, ".codex")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	swap := filepath.Join(configDir, "hooks.json.aav-swap")
	if err := os.WriteFile(swap, []byte(`{"description":"before"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	options := Options{ProjectRoot: root, HookBinary: writeFakeHook(t), DryRun: true}
	if _, err := NewManager().Install(context.Background(), options); err == nil {
		t.Fatal("dry-run should report pending recovery")
	}
	if string(mustRead(t, swap)) != `{"description":"before"}` {
		t.Fatal("dry-run changed interrupted swap")
	}
	status, err := NewManager().Status(context.Background(), Options{ProjectRoot: root})
	if err != nil || status.Status != "partial" {
		t.Fatalf("status = %#v, err = %v", status, err)
	}
	if string(mustRead(t, swap)) != `{"description":"before"}` {
		t.Fatal("status changed interrupted swap")
	}
}

func TestInstallRecoversMissingBackupFromTransactionState(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, ".codex")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	original := []byte(`{"description":"recover me"}`)
	configPath := filepath.Join(configDir, "hooks.json")
	if err := os.WriteFile(configPath, original, 0o600); err != nil {
		t.Fatal(err)
	}
	options := Options{ProjectRoot: root, HookBinary: writeFakeHook(t)}
	target, err := resolveTarget(options)
	if err != nil {
		t.Fatal(err)
	}
	interrupted := state{
		Version: stateVersion, InstallID: installID, Scope: ScopeProject, Phase: "installing",
		ConfigPath: target.configPath, BinaryPath: target.binaryPath, OriginalExists: true,
		OriginalSHA256: digest(original), BackupPath: target.backupPath, BinarySHA256: digest(mustRead(t, options.HookBinary)),
	}
	if err := writeState(target.statePath, interrupted); err != nil {
		t.Fatal(err)
	}
	result, err := NewManager().Install(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "installed" {
		t.Fatalf("result = %#v", result)
	}
	if !bytes.Equal(original, mustRead(t, target.backupPath)) {
		t.Fatal("backup was not recovered from untouched configuration")
	}
}

func TestInstallRefusesUnmanagedBackup(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, ".codex")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	original := []byte(`{"description":"do not adopt backup"}`)
	if err := os.WriteFile(filepath.Join(configDir, "hooks.json"), original, 0o600); err != nil {
		t.Fatal(err)
	}
	backupPath := filepath.Join(configDir, "hooks.json.aav-backup")
	if err := os.WriteFile(backupPath, original, 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := NewManager().Install(context.Background(), Options{ProjectRoot: root, HookBinary: writeFakeHook(t)})
	if err == nil {
		t.Fatal("expected unmanaged backup refusal")
	}
	assertMissing(t, filepath.Join(configDir, ".aav-codex-install.json"))
	assertMissing(t, result.BinaryPath)
	if !bytes.Equal(original, mustRead(t, backupPath)) {
		t.Fatal("unmanaged backup changed")
	}
}

func TestInstallRefusesManagedConfigurationSymlink(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside-hooks.json")
	original := []byte(`{"description":"outside must remain unchanged"}`)
	if err := os.WriteFile(outside, original, 0o600); err != nil {
		t.Fatal(err)
	}
	configDir := filepath.Join(root, ".codex")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "hooks.json")
	if err := os.Symlink(outside, configPath); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	if _, err := NewManager().Install(context.Background(), Options{ProjectRoot: root, HookBinary: writeFakeHook(t)}); err == nil {
		t.Fatal("installer followed a managed configuration symlink")
	}
	if !bytes.Equal(original, mustRead(t, outside)) {
		t.Fatal("installer changed the symlink target")
	}
	assertMissing(t, filepath.Join(configDir, ".aav-codex-install.json"))
}

func TestInstallAndUninstallRefuseUnmanagedBinary(t *testing.T) {
	root := t.TempDir()
	options := Options{ProjectRoot: root, HookBinary: writeFakeHook(t)}
	target, err := resolveTarget(options)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(target.binaryPath), 0o700); err != nil {
		t.Fatal(err)
	}
	unmanaged := []byte("unmanaged executable")
	if err := os.WriteFile(target.binaryPath, unmanaged, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := NewManager().Install(context.Background(), options); err == nil {
		t.Fatal("expected install to refuse unmanaged binary")
	}
	if _, err := NewManager().Uninstall(context.Background(), options); err == nil {
		t.Fatal("expected uninstall to refuse unmanaged binary")
	}
	if !bytes.Equal(unmanaged, mustRead(t, target.binaryPath)) {
		t.Fatal("unmanaged binary was changed")
	}
	assertMissing(t, target.statePath)
	assertMissing(t, target.configPath)
}

func TestUninstallRefusesMissingBackupWithoutChangingFiles(t *testing.T) {
	root := t.TempDir()
	manager := NewManager()
	options := Options{ProjectRoot: root, HookBinary: writeFakeHook(t)}
	installed, err := manager.Install(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	statePayload := mustRead(t, filepath.Join(root, ".codex", ".aav-codex-install.json"))
	var state state
	if err := json.Unmarshal(statePayload, &state); err != nil {
		t.Fatal(err)
	}
	if !state.OriginalExists {
		// Add an original marker and a deliberately missing backup to exercise refusal.
		state.OriginalExists = true
		state.BackupPath = filepath.Join(root, ".codex", "missing-backup")
		state.OriginalSHA256 = digest([]byte("{}"))
		if err := writeState(filepath.Join(root, ".codex", ".aav-codex-install.json"), state); err != nil {
			t.Fatal(err)
		}
	}
	before := mustRead(t, installed.ConfigPath)
	if _, err := manager.Uninstall(context.Background(), options); err == nil {
		t.Fatal("expected missing backup error")
	}
	if !bytes.Equal(before, mustRead(t, installed.ConfigPath)) {
		t.Fatal("configuration changed despite missing backup")
	}
}

func TestUninstallRestoresOriginalWhenManagedConfigurationIsMissing(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, ".codex")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	original := []byte(`{"description":"restore after interruption"}`)
	configPath := filepath.Join(configDir, "hooks.json")
	if err := os.WriteFile(configPath, original, 0o600); err != nil {
		t.Fatal(err)
	}
	options := Options{ProjectRoot: root, HookBinary: writeFakeHook(t)}
	if _, err := NewManager().Install(context.Background(), options); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(configPath); err != nil {
		t.Fatal(err)
	}
	if _, err := NewManager().Uninstall(context.Background(), options); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(original, mustRead(t, configPath)) {
		t.Fatal("original configuration was not restored")
	}
}

func writeFakeHook(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "source hook"+testExecutableSuffix())
	if err := os.WriteFile(path, []byte("fake hook executable"), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

func mustStat(t *testing.T, path string) os.FileInfo {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info
}

func assertMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("%s exists or returned unexpected error: %v", path, err)
	}
}
