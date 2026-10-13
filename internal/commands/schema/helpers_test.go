package schema

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const testPayloadSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {"order_id": {"type": "string"}},
  "required": ["order_id"]
}`

func TestSchemaPayloadFromJSONWrapsUnderJSONSchema(t *testing.T) {
	payload, err := SchemaPayloadFromJSON("order-placed", testPayloadSchema)
	if err != nil {
		t.Fatalf("SchemaPayloadFromJSON returned error: %v", err)
	}

	if len(payload) != 1 {
		t.Fatalf("payload should only contain json_schema, got keys %v", keysOf(payload))
	}
	js, ok := payload["json_schema"].(map[string]any)
	if !ok {
		t.Fatalf("payload[json_schema] = %#v, want a JSON object", payload["json_schema"])
	}
	if _, has := js["$schema"]; has {
		t.Errorf("json_schema still contains $schema: %v", js)
	}
	if js["type"] != "object" {
		t.Errorf("json_schema.type = %v, want object", js["type"])
	}
	if _, has := js["properties"]; !has {
		t.Errorf("json_schema lost its properties: %v", js)
	}
}

// The --json value must produce the same json_schema as the equivalent
// payload_schema.json on disk.
func TestSchemaPayloadFromJSONMatchesFilePayload(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "schema.json"), []byte(`{"slug":"order-placed","name":"Order placed"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "payload_schema.json"), []byte(testPayloadSchema), 0o644); err != nil {
		t.Fatal(err)
	}

	fromFiles, err := ReadAndMergeSchemaFiles(dir, "order-placed")
	if err != nil {
		t.Fatalf("ReadAndMergeSchemaFiles returned error: %v", err)
	}
	fromJSON, err := SchemaPayloadFromJSON("order-placed", testPayloadSchema)
	if err != nil {
		t.Fatalf("SchemaPayloadFromJSON returned error: %v", err)
	}
	if !reflect.DeepEqual(fromFiles["json_schema"], fromJSON["json_schema"]) {
		t.Errorf("json_schema differs:\nfiles: %#v\n--json: %#v", fromFiles["json_schema"], fromJSON["json_schema"])
	}
}

func TestSchemaPayloadFromJSONRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr string
	}{
		{"not json", `{"type":`, "failed to parse --json"},
		{"null", `null`, "must be a JSON Schema object"},
		{"array", `[{"type":"object"}]`, "failed to parse --json"},
		{"non-object root", `{"type":"string"}`, "root 'type' must be 'object'"},
		{"invalid schema", `{"type":"object","properties":{"id":{"type":"strng"}}}`, "is not a valid JSON Schema"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := SchemaPayloadFromJSON("order-placed", tt.raw)
			if err == nil {
				t.Fatalf("expected an error for %s", tt.raw)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func keysOf(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
