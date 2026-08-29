package buildinfo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestCheckedInReleaseMetadataIsAligned(t *testing.T) {
	root := filepath.Join("..", "..")
	version, err := os.ReadFile(filepath.Join(root, "VERSION"))
	if err != nil {
		t.Fatal(err)
	}
	value := strings.TrimSpace(string(version))
	if !regexp.MustCompile(`^\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$`).MatchString(value) {
		t.Fatalf("VERSION is not SemVer: %q", value)
	}
	payload, err := os.ReadFile(filepath.Join(root, "apps", "desktop", "wails.json"))
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Info struct {
			ProductVersion string `json:"productVersion"`
		} `json:"info"`
	}
	if err := json.Unmarshal(payload, &config); err != nil {
		t.Fatal(err)
	}
	if config.Info.ProductVersion != value {
		t.Fatalf("wails productVersion = %q, VERSION = %q", config.Info.ProductVersion, value)
	}

	windowsPayload, err := os.ReadFile(filepath.Join(root, "apps", "desktop", "build", "windows", "info.json"))
	if err != nil {
		t.Fatal(err)
	}
	var windowsInfo struct {
		Fixed map[string]string            `json:"fixed"`
		Info  map[string]map[string]string `json:"info"`
	}
	if err := json.Unmarshal(windowsPayload, &windowsInfo); err != nil {
		t.Fatal(err)
	}
	const productVersionTemplate = "{{.Info.ProductVersion}}"
	if windowsInfo.Fixed["file_version"] != productVersionTemplate || windowsInfo.Fixed["product_version"] != productVersionTemplate {
		t.Fatalf("Windows fixed version metadata is not aligned: %#v", windowsInfo.Fixed)
	}
	english, ok := windowsInfo.Info["0409"]
	if !ok || english["ProductVersion"] != productVersionTemplate || english["FileVersion"] != productVersionTemplate || english["ProductName"] != "{{.Info.ProductName}}" {
		t.Fatalf("Windows English version strings are not configured: %#v", windowsInfo.Info)
	}
}

func TestWindowsPackageIncludesStandaloneSetupTools(t *testing.T) {
	root := filepath.Join("..", "..")
	packageScript, err := os.ReadFile(filepath.Join(root, "scripts", "package.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	installerScript, err := os.ReadFile(filepath.Join(root, "apps", "desktop", "build", "windows", "installer", "project.nsi"))
	if err != nil {
		t.Fatal(err)
	}

	for _, companion := range []string{"aav.exe", "aav-codex-hook.exe", "aav-wrapper.exe"} {
		if !strings.Contains(string(packageScript), companion) {
			t.Fatalf("portable package script does not include %s", companion)
		}
		if !strings.Contains(string(installerScript), `"package-tools\`+companion+`"`) {
			t.Fatalf("NSIS installer does not include %s", companion)
		}
	}

	for _, fragment := range []string{
		"!define REQUEST_EXECUTION_LEVEL \"user\"",
		"InstallDir \"$LOCALAPPDATA\\Programs\\${INFO_COMPANYNAME}\\${INFO_PRODUCTNAME}\"",
		"InstallDirRegKey HKCU \"${UNINST_KEY}\" \"InstallLocation\"",
		"WriteRegStr HKCU \"${UNINST_KEY}\" \"InstallLocation\" \"$INSTDIR\"",
		"DeleteRegKey HKCU \"${UNINST_KEY}\"",
	} {
		if !strings.Contains(string(installerScript), fragment) {
			t.Fatalf("NSIS installer is missing per-user configuration %q", fragment)
		}
	}
	if strings.Contains(string(installerScript), "RMDir /r \"$AppData\\${PRODUCT_EXECUTABLE}\"") {
		t.Fatal("NSIS uninstall must preserve local application data")
	}

	lifecycleScript, err := os.ReadFile(filepath.Join(root, "scripts", "test-windows-installer.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"action=install", "action=launch", "action=uninstall", "installer_lifecycle=passed"} {
		if !strings.Contains(string(lifecycleScript), action) {
			t.Fatalf("installer lifecycle script is missing %q", action)
		}
	}
}
