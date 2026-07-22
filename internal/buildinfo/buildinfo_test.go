package buildinfo

import "testing"

func TestIdentityIsStable(t *testing.T) {
	if Name != "agent-action-visualizer" {
		t.Fatalf("unexpected application name %q", Name)
	}
	if Version == "" {
		t.Fatal("version must not be empty")
	}
}
