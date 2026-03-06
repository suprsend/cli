package cli_test

import (
	"net/http"
	"strings"
	"testing"
)

// TestCategoryCommit_Success verifies that commit sends a PATCH to the commit endpoint and exits 0.
func TestCategoryCommit_Success(t *testing.T) {
	var capturedMethod, capturedPath string
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	})

	// Use a temp dir with no translation subdir so PushTranslations silently skips.
	tmpDir := t.TempDir()

	stdout, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "category", "commit", "--workspace", "test-ws", "--dir", tmpDir)

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout: %s", exitCode, stdout)
	}
	if capturedMethod != http.MethodPatch {
		t.Errorf("expected PATCH, got %s", capturedMethod)
	}
	if !strings.Contains(capturedPath, "preference_category/commit") {
		t.Errorf("unexpected path: %s", capturedPath)
	}
}

// TestCategoryCommit_WithCommitMessage verifies that --commit-message is sent as a query param.
func TestCategoryCommit_WithCommitMessage(t *testing.T) {
	var capturedCommitMessage string
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedCommitMessage = r.URL.Query().Get("commit_message")
		w.WriteHeader(http.StatusOK)
	})

	tmpDir := t.TempDir()

	stdout, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "category", "commit", "--workspace", "test-ws", "--dir", tmpDir, "--commit-message", "release-v2")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout: %s", exitCode, stdout)
	}
	if capturedCommitMessage != "release-v2" {
		t.Errorf("expected commit_message='release-v2', got: %q", capturedCommitMessage)
	}
}

// TestCategoryCommit_APIError verifies non-zero exit on API error response.
func TestCategoryCommit_APIError(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"code": 401, "message": "Invalid service token"}`))
	})

	tmpDir := t.TempDir()

	_, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "bad-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "category", "commit", "--workspace", "test-ws", "--dir", tmpDir)

	if exitCode == 0 {
		t.Error("expected non-zero exit code for API error, got 0")
	}
}
