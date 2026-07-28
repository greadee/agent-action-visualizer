package codex

import "testing"

func TestPatchCountsMarkerLikeContent(t *testing.T) {
	evidence := parsePatch("*** Begin Patch\n*** Update File: markers.txt\n@@\n---removed\n+++added\n*** End Patch")
	if len(evidence) != 1 || evidence[0].linesAdded == nil || *evidence[0].linesAdded != 1 || evidence[0].linesDeleted == nil || *evidence[0].linesDeleted != 1 {
		t.Fatalf("marker-like content delta = %#v", evidence)
	}
}
