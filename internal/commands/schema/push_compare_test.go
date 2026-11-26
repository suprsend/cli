package schema

import "testing"

func TestSchemaFingerprint(t *testing.T) {
	remote := map[string]any{
		"slug": "order", "name": "Order", "description": nil, "is_enabled": true,
		"json_schema": map[string]any{"type": "object", "$schema": "https://json-schema.org/draft/2020-12/schema"},
		"version_no":  3,
	}
	local := map[string]any{"slug": "order", "name": "Order", "description": "", "json_schema": map[string]any{"type": "object"}}
	if fingerprint(local) != fingerprint(remote) {
		t.Error("a pulled schema should match: $schema, a null description and fields the API manages don't count, and a missing is_enabled means enabled")
	}
	remote["is_enabled"] = false
	if fingerprint(local) == fingerprint(remote) {
		t.Error("pushing without is_enabled enables a disabled schema, so it differs")
	}
	local["is_enabled"] = false
	local["json_schema"] = map[string]any{"type": "object", "required": []any{"id"}}
	if fingerprint(local) == fingerprint(remote) {
		t.Error("a changed JSON Schema should differ")
	}
}

// The API adds additionalProperties:false and keeps a name the request
// leaves out; neither is a change.
func TestSchemaSameAs(t *testing.T) {
	remote := map[string]any{"name": "Order", "description": "d", "json_schema": map[string]any{
		"type": "object", "properties": map[string]any{"id": map[string]any{"type": "string"}}, "additionalProperties": false}}
	jsonOnly := map[string]any{"json_schema": map[string]any{
		"type": "object", "properties": map[string]any{"id": map[string]any{"type": "string"}}}}
	if !sameAs(jsonOnly, remote) {
		t.Error("a payload without name or additionalProperties should match the stored schema")
	}
	if sameAs(map[string]any{"name": "Orders", "json_schema": jsonOnly["json_schema"]}, remote) {
		t.Error("a renamed schema should differ")
	}
}
