package buildinfo

import "testing"

func TestIdentityIsStable(t *testing.T) {
	if Name != "agent-action-visualizer" {
		t.Fatalf("unexpected application name %q", Name)
	}
	if Version == "" {
		t.Fatal("version must not be empty")
	}
	if Commit == "" || BuildDate == "" {
		t.Fatalf("release identity must not be empty: %+v", Current())
	}
	if info := Current(); info.Name != Name || info.Version != Version {
		t.Fatalf("current build info = %+v", info)
	}
}
