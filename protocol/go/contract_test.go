package protocol

import (
	"bytes"
	"os"
	"testing"
)

func TestGoSchemaAndTypeScriptEnumsAgree(t *testing.T) {
	schema, err := os.ReadFile("../schema/v1/event.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	typescript, err := os.ReadFile("../typescript/event.ts")
	if err != nil {
		t.Fatal(err)
	}

	assertShared := func(value string) {
		t.Helper()
		quoted := []byte(`"` + value + `"`)
		if !bytes.Contains(schema, quoted) {
			t.Errorf("schema is missing %q", value)
		}
		if !bytes.Contains(typescript, []byte(`'`+value+`'`)) && !bytes.Contains(typescript, quoted) {
			t.Errorf("TypeScript is missing %q", value)
		}
	}
	for value := range validEventTypes {
		assertShared(string(value))
	}
	for value := range validSources {
		assertShared(string(value))
	}
	for value := range validConfidences {
		assertShared(string(value))
	}
}
