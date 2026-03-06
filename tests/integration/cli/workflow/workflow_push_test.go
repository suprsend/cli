package cli_test

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupWorkflowDir creates a temp dir with a workflow JSON file.
func setupWorkflowDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	data := readTestdata(t, "workflows", "sample_workflow.json")
	if err := os.WriteFile(filepath.Join(tmpDir, "test-workflow.json"), data, 0644); err != nil {
		t.Fatalf("failed to write workflow file: %v", err)
	}
	return tmpDir
}

func TestWorkflowPush_CommitTrue(t *testing.T) {
	pushResponse := readTestdata(t, "workflows", "workflow_push_response.json")
	workflowDir := setupWorkflowDir(t)

	var capturedMethod, capturedPath, capturedCommit string
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		capturedCommit = r.URL.Query().Get("commit")
		w.Header().Set("Content-Type", "application/json")
		w.Write(pushResponse)
	})

	stdout, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "workflow", "push", "--workspace", "test-ws", "--dir", workflowDir, "--commit", "true")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout: %s", exitCode, stdout)
	}
	if capturedMethod != http.MethodPost {
		t.Errorf("expected POST, got %s", capturedMethod)
	}
	if !strings.Contains(capturedPath, "workflow") {
		t.Errorf("unexpected path: %s", capturedPath)
	}
	if capturedCommit != "true" {
		t.Errorf("expected commit=true, got: %s", capturedCommit)
	}
	if !strings.Contains(stdout, "Successfully pushed: 1") {
		t.Errorf("expected 'Successfully pushed: 1' in stdout, got: %s", stdout)
	}
}

func TestWorkflowPush_CommitFalse(t *testing.T) {
	pushResponse := readTestdata(t, "workflows", "workflow_push_response.json")
	workflowDir := setupWorkflowDir(t)

	var capturedCommit string
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedCommit = r.URL.Query().Get("commit")
		w.Header().Set("Content-Type", "application/json")
		w.Write(pushResponse)
	})

	stdout, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "workflow", "push", "--workspace", "test-ws", "--dir", workflowDir, "--commit", "false")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout: %s", exitCode, stdout)
	}
	if capturedCommit != "false" {
		t.Errorf("expected commit=false, got: %s", capturedCommit)
	}
}

func TestWorkflowPush_BySlug(t *testing.T) {
	pushResponse := readTestdata(t, "workflows", "workflow_push_response.json")
	workflowDir := setupWorkflowDir(t)

	var capturedPath string
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write(pushResponse)
	})

	stdout, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "workflow", "push", "--workspace", "test-ws", "--dir", workflowDir, "--slug", "test-workflow")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout: %s", exitCode, stdout)
	}
	if !strings.Contains(capturedPath, "test-workflow") {
		t.Errorf("expected slug in path, got: %s", capturedPath)
	}
}

func TestWorkflowPush_MissingDir(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not be called when dir is missing")
	})

	_, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "workflow", "push", "--workspace", "test-ws", "--dir", "/nonexistent/path/abc123")

	if exitCode == 0 {
		t.Error("expected non-zero exit code when dir is missing, got 0")
	}
}

func TestWorkflowPush_APIError(t *testing.T) {
	workflowDir := setupWorkflowDir(t)

	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"code": 401, "message": "Invalid service token"}`))
	})

	stdout, _, _ := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "bad-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "workflow", "push", "--workspace", "test-ws", "--dir", workflowDir)

	// Push command reports failure in summary but exits 0 (errors are tracked in stats)
	if !strings.Contains(stdout, "Failed to push: 1") {
		t.Errorf("expected 'Failed to push: 1' in stdout, got: %s", stdout)
	}
}
