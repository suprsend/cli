package cli_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestTranslationCommit_Success(t *testing.T) {
	var capturedMethod, capturedPath string
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	})

	stdout, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "translation", "commit", "--workspace", "test-ws")

	// translation commit uses Run: not RunE: so exits 0 always
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout: %s", exitCode, stdout)
	}
	if capturedMethod != http.MethodPatch {
		t.Errorf("expected PATCH, got %s", capturedMethod)
	}
	if !strings.Contains(capturedPath, "translation") || !strings.Contains(capturedPath, "commit") {
		t.Errorf("unexpected path: %s", capturedPath)
	}
	if !strings.Contains(stdout, "Successfully committed translation") {
		t.Errorf("expected success message in stdout, got: %s", stdout)
	}
}

func TestTranslationCommit_WithCommitMessage(t *testing.T) {
	var capturedCommitMessage string
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedCommitMessage = r.URL.Query().Get("commit_message")
		w.WriteHeader(http.StatusOK)
	})

	stdout, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "translation", "commit", "--workspace", "test-ws", "--commit-message", "release-v3")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout: %s", exitCode, stdout)
	}
	if capturedCommitMessage != "release-v3" {
		t.Errorf("expected commit_message='release-v3', got: %q", capturedCommitMessage)
	}
}

func TestTranslationCommit_APIError(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"code": 401, "message": "Invalid service token"}`))
	})

	stdout, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "bad-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "translation", "commit", "--workspace", "test-ws")

	// translation commit uses Run: not RunE: - exits 0 even on error
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0 (Run: not RunE:); stdout: %s", exitCode, stdout)
	}
	// On API error, success message should NOT appear in stdout
	if strings.Contains(stdout, "Successfully committed translation") {
		t.Errorf("unexpected success message in stdout on API error: %s", stdout)
	}
}
