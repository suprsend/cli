package unit_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/suprsend/cli/internal/commands/category"
	"github.com/suprsend/cli/internal/commands/event"
	"github.com/suprsend/cli/internal/commands/workflow"
	"github.com/suprsend/cli/mgmnt"
)

func TestWriteWorkflowsToFiles(t *testing.T) {
	tmpDir := t.TempDir()
	resp := mgmnt.WorkflowsResponse{
		Results: []any{
			map[string]any{"slug": "wf-alpha", "name": "Alpha"},
			map[string]any{"slug": "wf-beta", "name": "Beta"},
		},
	}
	stats, err := workflow.WriteWorkflowsToFiles(resp, tmpDir)
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
	for _, slug := range []string{"wf-alpha", "wf-beta"} {
		path := filepath.Join(tmpDir, slug+".json")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("file %s should exist", path)
		}
	}
}

func TestWriteWorkflowsToFiles_InvalidFormat(t *testing.T) {
	tmpDir := t.TempDir()
	resp := mgmnt.WorkflowsResponse{
		Results: []any{"not a map"},
	}
	stats, err := workflow.WriteWorkflowsToFiles(resp, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.Failed != 1 {
		t.Errorf("Failed = %d, want 1", stats.Failed)
	}
}

func TestWriteEventsToFiles(t *testing.T) {
	tmpDir := t.TempDir()
	resp := &mgmnt.EventsResponse{
		Results: []any{
			map[string]any{"name": "event-1", "schema": "s1"},
			map[string]any{"name": "event-2", "schema": "s2"},
		},
	}
	stats, err := event.WriteEventsToFiles(resp, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.Success != 2 {
		t.Errorf("Success = %d, want 2", stats.Success)
	}
	// Verify event_schema_mapping.json was created
	mappingPath := filepath.Join(tmpDir, "event_schema_mapping.json")
	if _, err := os.Stat(mappingPath); os.IsNotExist(err) {
		t.Error("event_schema_mapping.json should exist")
	}
	// Verify contents
	data, _ := os.ReadFile(mappingPath)
	var mapping map[string]any
	json.Unmarshal(data, &mapping)
	events, ok := mapping["events"].([]any)
	if !ok {
		t.Fatal("events key should be an array")
	}
	if len(events) != 2 {
		t.Errorf("events count = %d, want 2", len(events))
	}
}

func TestCategoryWriteToFileWithPath(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "sub", "categories.json")

	data := map[string]any{
		"root_categories": []any{
			map[string]any{"name": "cat-1"},
		},
	}

	err := category.WriteToFileWithPath(data, filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file exists (including subdirectory creation)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("file should exist")
	}

	// Verify valid JSON
	content, _ := os.ReadFile(filePath)
	var result map[string]any
	if err := json.Unmarshal(content, &result); err != nil {
		t.Fatalf("file content is not valid JSON: %v", err)
	}
}

func TestCategoryReadFromFile_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.json")
	data := map[string]any{"key": "value"}
	b, _ := json.Marshal(data)
	os.WriteFile(filePath, b, 0644)

	result, err := category.ReadFromFile(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatal("result should be a map")
	}
	if m["key"] != "value" {
		t.Errorf("key = %v, want 'value'", m["key"])
	}
}

func TestCategoryReadFromFile_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "invalid.json")
	os.WriteFile(filePath, []byte("not json"), 0644)

	_, err := category.ReadFromFile(filePath)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestCategoryReadFromFile_MissingFile(t *testing.T) {
	_, err := category.ReadFromFile("/nonexistent/file.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
