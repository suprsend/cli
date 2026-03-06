package cli_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestWorkflowGet_Success(t *testing.T) {
	getResponse := readTestdata(t, "workflows", "workflow_get_response.json")

	var capturedMethod, capturedPath, capturedMode string
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		capturedMode = r.URL.Query().Get("mode")
		w.Header().Set("Content-Type", "application/json")
		w.Write(getResponse)
	})

	stdout, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "workflow", "get", "test-workflow", "--workspace", "test-ws")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout: %s", exitCode, stdout)
	}
	if capturedMethod != http.MethodGet {
		t.Errorf("expected GET, got %s", capturedMethod)
	}
	if !strings.Contains(capturedPath, "test-workflow") {
		t.Errorf("expected slug in path, got: %s", capturedPath)
	}
	if capturedMode != "live" {
		t.Errorf("expected default mode=live, got: %s", capturedMode)
	}
	if !strings.Contains(stdout, "test-workflow") {
		t.Errorf("expected slug in stdout, got: %s", stdout)
	}
}

func TestWorkflowGet_MissingSlug(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not be called when slug is missing")
	})

	_, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "workflow", "get", "--workspace", "test-ws")

	if exitCode == 0 {
		t.Error("expected non-zero exit code when slug is missing, got 0")
	}
}

func TestWorkflowGet_APIError(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"code": 404, "message": "Workflow not found"}`))
	})

	_, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "workflow", "get", "nonexistent-slug", "--workspace", "test-ws")

	if exitCode == 0 {
		t.Error("expected non-zero exit code for API error, got 0")
	}
}
