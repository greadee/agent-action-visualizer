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
