package cli_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestSchemaCommit_Success(t *testing.T) {
	var capturedMethod, capturedPath string
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	})

	stdout, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "schema", "commit", "test-schema", "--workspace", "test-ws")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout: %s", exitCode, stdout)
	}
	if capturedMethod != http.MethodPatch {
		t.Errorf("expected PATCH, got %s", capturedMethod)
	}
	if !strings.Contains(capturedPath, "test-schema") || !strings.Contains(capturedPath, "commit") {
		t.Errorf("unexpected path: %s", capturedPath)
	}
}

func TestSchemaCommit_WithCommitMessage(t *testing.T) {
	var capturedCommitMessage string
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedCommitMessage = r.URL.Query().Get("commit_message")
		w.WriteHeader(http.StatusOK)
	})

	stdout, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "schema", "commit", "test-schema", "--workspace", "test-ws", "--commit-message", "release-v2")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout: %s", exitCode, stdout)
	}
	if capturedCommitMessage != "release-v2" {
		t.Errorf("expected commit_message='release-v2', got: %q", capturedCommitMessage)
	}
}

func TestSchemaCommit_MissingSlug(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not be called when slug is missing")
	})

	_, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "schema", "commit", "--workspace", "test-ws")

	if exitCode == 0 {
		t.Error("expected non-zero exit code when slug is missing, got 0")
	}
}

func TestSchemaCommit_APIError(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"code": 401, "message": "Invalid service token"}`))
	})

	_, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "bad-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "schema", "commit", "test-schema", "--workspace", "test-ws")

	if exitCode == 0 {
		t.Error("expected non-zero exit code for API error, got 0")
	}
}
