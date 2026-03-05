package unit_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/suprsend/cli/internal/commands/schema"
	"github.com/suprsend/cli/mgmnt"
)

func TestMergeJSONSchemas_DisjointProperties(t *testing.T) {
	base := map[string]any{
		"type":       "object",
		"properties": map[string]any{"name": map[string]any{"type": "string"}},
	}
	add := map[string]any{
		"properties": map[string]any{"email": map[string]any{"type": "string"}},
	}
	result := schema.MergeJSONSchemas(base, add)
	props, ok := result["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties should be a map")
	}
	if _, ok := props["name"]; !ok {
		t.Error("missing 'name' in merged properties")
	}
	if _, ok := props["email"]; !ok {
		t.Error("missing 'email' in merged properties")
	}
}

func TestMergeJSONSchemas_RequiredUnion(t *testing.T) {
	base := map[string]any{
		"required": []any{"name", "age"},
	}
	add := map[string]any{
		"required": []any{"age", "email"},
	}
	result := schema.MergeJSONSchemas(base, add)
	req, ok := result["required"].([]any)
	if !ok {
		t.Fatal("required should be []any")
	}
	// Should be union: name, age, email (no duplicates)
	strs := make(map[string]bool)
	for _, v := range req {
		strs[v.(string)] = true
	}
	if len(strs) != 3 {
		t.Errorf("expected 3 unique required fields, got %d: %v", len(strs), req)
	}
	for _, expected := range []string{"name", "age", "email"} {
		if !strs[expected] {
			t.Errorf("missing required field: %s", expected)
		}
	}
}

func TestMergeJSONSchemas_NestedRecursive(t *testing.T) {
	base := map[string]any{
		"properties": map[string]any{
			"address": map[string]any{
				"type":       "object",
				"properties": map[string]any{"city": map[string]any{"type": "string"}},
			},
		},
	}
	add := map[string]any{
		"properties": map[string]any{
			"address": map[string]any{
				"properties": map[string]any{"zip": map[string]any{"type": "string"}},
			},
		},
	}
	result := schema.MergeJSONSchemas(base, add)
	props := result["properties"].(map[string]any)
	addr := props["address"].(map[string]any)
	addrProps := addr["properties"].(map[string]any)
	if _, ok := addrProps["city"]; !ok {
		t.Error("missing 'city' in nested merge")
	}
	if _, ok := addrProps["zip"]; !ok {
		t.Error("missing 'zip' in nested merge")
	}
}

func TestMergeJSONSchemas_TypePreservation(t *testing.T) {
	base := map[string]any{"type": "object"}
	add := map[string]any{"type": "array"}
	result := schema.MergeJSONSchemas(base, add)
	if result["type"] != "object" {
		t.Errorf("type = %v, want 'object' (base should win)", result["type"])
	}
}

func TestMergeJSONSchemas_AllOfConcat(t *testing.T) {
	base := map[string]any{
		"allOf": []any{map[string]any{"type": "string"}},
	}
	add := map[string]any{
		"allOf": []any{map[string]any{"minLength": float64(1)}},
	}
	result := schema.MergeJSONSchemas(base, add)
	allOf, ok := result["allOf"].([]any)
	if !ok {
		t.Fatal("allOf should be []any")
	}
	if len(allOf) != 2 {
		t.Errorf("allOf length = %d, want 2", len(allOf))
	}
}

func TestMergeJSONSchemas_DefseMerge(t *testing.T) {
	base := map[string]any{
		"$defs": map[string]any{"Address": map[string]any{"type": "object"}},
	}
	add := map[string]any{
		"$defs": map[string]any{"Phone": map[string]any{"type": "string"}},
	}
	result := schema.MergeJSONSchemas(base, add)
	defs, ok := result["$defs"].(map[string]any)
	if !ok {
		t.Fatal("$defs should be a map")
	}
	if _, ok := defs["Address"]; !ok {
		t.Error("missing 'Address' in $defs")
	}
	if _, ok := defs["Phone"]; !ok {
		t.Error("missing 'Phone' in $defs")
	}
}

func TestMergeJSONSchemas_EmptyBase(t *testing.T) {
	base := map[string]any{}
	add := map[string]any{
		"type":       "object",
		"properties": map[string]any{"name": map[string]any{"type": "string"}},
	}
	result := schema.MergeJSONSchemas(base, add)
	if result["type"] != "object" {
		t.Errorf("type = %v, want 'object'", result["type"])
	}
}

func TestMergeJSONSchemas_EmptyAdd(t *testing.T) {
	base := map[string]any{
		"type":       "object",
		"properties": map[string]any{"name": map[string]any{"type": "string"}},
	}
	add := map[string]any{}
	result := schema.MergeJSONSchemas(base, add)
	if result["type"] != "object" {
		t.Errorf("type = %v, want 'object'", result["type"])
	}
}

func TestMergeAndValidate_Success(t *testing.T) {
	base := `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {"name": {"type": "string"}},
		"required": ["name"]
	}`
	patch := map[string]any{
		"properties": map[string]any{"email": map[string]any{"type": "string"}},
		"required":   []any{"email"},
	}
	result, err := schema.MergeAndValidate(base, patch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Verify it's valid JSON
	var m map[string]any
	if err := json.Unmarshal(result, &m); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}
	props := m["properties"].(map[string]any)
	if _, ok := props["name"]; !ok {
		t.Error("missing 'name' in merged result")
	}
	if _, ok := props["email"]; !ok {
		t.Error("missing 'email' in merged result")
	}
}

func TestMergeAndValidate_InvalidBaseJSON(t *testing.T) {
	_, err := schema.MergeAndValidate("not valid json", map[string]any{})
	if err == nil {
		t.Fatal("expected error for invalid base JSON")
	}
}

func TestMergeUnderDataAndValidate_Success(t *testing.T) {
	base := `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"data": {
				"type": "object",
				"properties": {"existing": {"type": "string"}}
			}
		}
	}`
	d := map[string]any{
		"properties": map[string]any{"new_field": map[string]any{"type": "integer"}},
	}
	result, err := schema.MergeUnderDataAndValidate(base, d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var m map[string]any
	json.Unmarshal(result, &m)
	props := m["properties"].(map[string]any)
	data := props["data"].(map[string]any)
	dataProps := data["properties"].(map[string]any)
	if _, ok := dataProps["existing"]; !ok {
		t.Error("missing 'existing' in data properties")
	}
	if _, ok := dataProps["new_field"]; !ok {
		t.Error("missing 'new_field' in data properties")
	}
}

func TestMergeUnderDataAndValidate_DefsHoisted(t *testing.T) {
	base := `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {"data": {"type": "object", "properties": {}}}
	}`
	d := map[string]any{
		"$defs": map[string]any{"MyType": map[string]any{"type": "string"}},
		"properties": map[string]any{
			"field": map[string]any{"$ref": "#/$defs/MyType"},
		},
	}
	result, err := schema.MergeUnderDataAndValidate(base, d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var m map[string]any
	json.Unmarshal(result, &m)
	defs, ok := m["$defs"].(map[string]any)
	if !ok {
		t.Fatal("$defs should be hoisted to root")
	}
	if _, ok := defs["MyType"]; !ok {
		t.Error("missing 'MyType' in root $defs")
	}
}

func TestMergeUnderDataAndValidate_LegacyDefinitions(t *testing.T) {
	base := `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {"data": {"type": "object", "properties": {}}}
	}`
	d := map[string]any{
		"definitions": map[string]any{"OldType": map[string]any{"type": "integer"}},
	}
	result, err := schema.MergeUnderDataAndValidate(base, d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var m map[string]any
	json.Unmarshal(result, &m)
	defs, ok := m["$defs"].(map[string]any)
	if !ok {
		t.Fatal("legacy definitions should be hoisted to $defs")
	}
	if _, ok := defs["OldType"]; !ok {
		t.Error("missing 'OldType' in root $defs")
	}
}

func TestWriteSchemasToFiles(t *testing.T) {
	tmpDir := t.TempDir()
	resp := &mgmnt.SchemasResponse{
		Results: []any{
			map[string]any{"slug": "schema-a", "title": "Schema A"},
			map[string]any{"slug": "schema-b", "title": "Schema B"},
		},
	}
	stats, err := schema.WriteSchemasToFiles(resp, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.Total != 2 {
		t.Errorf("Total = %d, want 2", stats.Total)
	}
	if stats.Success != 2 {
		t.Errorf("Success = %d, want 2", stats.Success)
	}
	// Verify files exist
	for _, slug := range []string{"schema-a", "schema-b"} {
		path := filepath.Join(tmpDir, slug+".json")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("file %s should exist", path)
		}
	}
}

func TestWriteSchemasToFiles_InvalidFormat(t *testing.T) {
	tmpDir := t.TempDir()
	resp := &mgmnt.SchemasResponse{
		Results: []any{
			"not a map", // invalid format
		},
	}
	stats, err := schema.WriteSchemasToFiles(resp, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.Failed != 1 {
		t.Errorf("Failed = %d, want 1", stats.Failed)
	}
}
