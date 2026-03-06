package cli_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestWorkflowEnable_Success(t *testing.T) {
	var capturedMethod, capturedPath string
	var capturedBody map[string]any
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		w.WriteHeader(http.StatusOK)
	})

	stdout, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "workflow", "enable", "my-workflow", "--workspace", "test-ws")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout: %s", exitCode, stdout)
	}
	if capturedMethod != http.MethodPatch {
		t.Errorf("expected PATCH, got %s", capturedMethod)
	}
	if !strings.Contains(capturedPath, "my-workflow") || !strings.Contains(capturedPath, "enable") {
		t.Errorf("unexpected path: %s", capturedPath)
	}
	if isEnabled, ok := capturedBody["is_enabled"].(bool); !ok || !isEnabled {
		t.Errorf("expected is_enabled=true in request body, got: %v", capturedBody)
	}
	if !strings.Contains(stdout, "Enabled workflow: my-workflow") {
		t.Errorf("expected success message in stdout, got: %s", stdout)
	}
}

func TestWorkflowDisable_Success(t *testing.T) {
	var capturedBody map[string]any
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		w.WriteHeader(http.StatusOK)
	})

	stdout, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "workflow", "disable", "my-workflow", "--workspace", "test-ws")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout: %s", exitCode, stdout)
	}
	if isEnabled, ok := capturedBody["is_enabled"].(bool); !ok || isEnabled {
		t.Errorf("expected is_enabled=false in request body, got: %v", capturedBody)
	}
	if !strings.Contains(stdout, "Disabled workflow: my-workflow") {
		t.Errorf("expected success message in stdout, got: %s", stdout)
	}
}

func TestWorkflowEnable_MissingSlug(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not be called when slug is missing")
	})

	_, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "workflow", "enable", "--workspace", "test-ws")

	if exitCode == 0 {
		t.Error("expected non-zero exit code when slug is missing, got 0")
	}
}

func TestWorkflowEnable_APIError(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"code": 404, "message": "Workflow not found"}`))
	})

	_, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "workflow", "enable", "nonexistent-slug", "--workspace", "test-ws")

	if exitCode == 0 {
		t.Error("expected non-zero exit code for API error, got 0")
	}
}
