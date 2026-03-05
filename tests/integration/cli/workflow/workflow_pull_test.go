package cli_test

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestWorkflowPull_All(t *testing.T) {
	listData, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "workflows", "workflow_list_response.json"))
	if err != nil {
		t.Fatalf("failed to read workflow_list_response.json: %v", err)
	}

	var reqCount int32
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&reqCount, 1)
		w.Header().Set("Content-Type", "application/json")
		if count == 1 {
			w.Write(listData)
		} else {
			w.Write([]byte(`{"results":[],"meta":{"count":0,"limit":50,"offset":50}}`))
		}
	})

	tmpDir := t.TempDir()
	stdout, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "workflow", "pull", "--workspace", "test-ws", "--dir", tmpDir, "--force")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout: %s", exitCode, stdout)
	}
	if !strings.Contains(stdout, "Total workflows processed: 2") {
		t.Errorf("expected 'Total workflows processed: 2' in stdout, got: %s", stdout)
	}
	if !strings.Contains(stdout, "Successfully updated: 2") {
		t.Errorf("expected 'Successfully updated: 2' in stdout, got: %s", stdout)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "workflow-one.json")); err != nil {
		t.Errorf("expected workflow-one.json to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "workflow-two.json")); err != nil {
		t.Errorf("expected workflow-two.json to exist: %v", err)
	}
}

func TestWorkflowPull_BySlug(t *testing.T) {
	sampleData, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "workflows", "sample_workflow.json"))
	if err != nil {
		t.Fatalf("failed to read sample_workflow.json: %v", err)
	}

	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(sampleData)
	})

	tmpDir := t.TempDir()
	stdout, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "workflow", "pull", "--workspace", "test-ws", "--slug", "test-workflow", "--dir", tmpDir, "--force")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout: %s", exitCode, stdout)
	}
	outFile := filepath.Join(tmpDir, "test-workflow.json")
	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("expected test-workflow.json to exist: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal test-workflow.json: %v", err)
	}
	if result["slug"] != "test-workflow" {
		t.Errorf("slug = %v, want test-workflow", result["slug"])
	}
}
