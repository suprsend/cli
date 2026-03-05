package unit_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/suprsend/cli/internal/utils"
)

// captureStdout captures stdout output during function execution.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	buf := make([]byte, 64*1024)
	n, _ := r.Read(buf)
	return string(buf[:n])
}

type testRow struct {
	Name   string `json:"name" yaml:"name"`
	Status string `json:"status" yaml:"status"`
}

func TestOutputData_JSON(t *testing.T) {
	data := []testRow{
		{Name: "test-1", Status: "active"},
		{Name: "test-2", Status: "inactive"},
	}
	output := captureStdout(t, func() {
		utils.OutputData(data, "json")
	})
	// Strip ANSI codes for comparison - the output might be colorized
	output = strings.TrimSpace(output)
	// Try to parse as JSON (may have ANSI escape codes if color is on)
	// Find the JSON array in the output
	start := strings.Index(output, "[")
	end := strings.LastIndex(output, "]")
	if start >= 0 && end > start {
		jsonStr := output[start : end+1]
		var result []testRow
		if err := json.Unmarshal([]byte(jsonStr), &result); err == nil {
			if len(result) != 2 {
				t.Errorf("expected 2 items, got %d", len(result))
			}
		}
	}
	// At minimum, output should contain the data
	if !strings.Contains(output, "test-1") {
		t.Errorf("output should contain 'test-1', got: %s", output)
	}
}

func TestOutputData_YAML(t *testing.T) {
	data := []testRow{
		{Name: "test-1", Status: "active"},
	}
	output := captureStdout(t, func() {
		utils.OutputData(data, "yaml")
	})
	if !strings.Contains(output, "test-1") {
		t.Errorf("output should contain 'test-1', got: %s", output)
	}
	if !strings.Contains(output, "name") {
		t.Errorf("output should contain 'name' key, got: %s", output)
	}
}

func TestOutputData_Table(t *testing.T) {
	data := []testRow{
		{Name: "test-1", Status: "active"},
		{Name: "test-2", Status: "inactive"},
	}
	output := captureStdout(t, func() {
		utils.OutputData(data, "table")
	})
	if !strings.Contains(strings.ToUpper(output), "NAME") {
		t.Errorf("table output should contain header 'Name' (case-insensitive), got: %s", output)
	}
	if !strings.Contains(output, "test-1") {
		t.Errorf("table output should contain 'test-1', got: %s", output)
	}
}
