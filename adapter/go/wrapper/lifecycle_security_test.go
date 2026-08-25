package wrapper

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestCommandLabelIsSingleLineBoundedAndRedacted(t *testing.T) {
	label := commandLabel("tool\npassword=private-value\x00")
	if strings.ContainsAny(label, "\r\n\x00") || strings.Contains(label, "private-value") {
		t.Fatalf("unsafe command label %q", label)
	}
	label = commandLabel(strings.Repeat("\u754c", 80))
	if len(label) > 128 || !utf8.ValidString(label) {
		t.Fatalf("invalid bounded label %q", label)
	}
}
