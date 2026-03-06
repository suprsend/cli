package cli_test

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupTranslationDir creates a temp dir with a locale JSON file.
func setupTranslationDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	content := []byte(`{"hello": "Hello", "world": "World"}`)
	if err := os.WriteFile(filepath.Join(tmpDir, "en.json"), content, 0644); err != nil {
		t.Fatalf("failed to write translation file: %v", err)
	}
	return tmpDir
}

func TestTranslationPush_Success(t *testing.T) {
	translationDir := setupTranslationDir(t)

	var capturedMethod, capturedPath string
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	})

	stdout, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "translation", "push", "--workspace", "test-ws", "--dir", translationDir)

	// translation push uses Run: not RunE: so exits 0 always
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout: %s", exitCode, stdout)
	}
	if capturedMethod != http.MethodPost {
		t.Errorf("expected POST, got %s", capturedMethod)
	}
	if !strings.Contains(capturedPath, "translation") || !strings.Contains(capturedPath, "en.json") {
		t.Errorf("unexpected path: %s", capturedPath)
	}
	if !strings.Contains(stdout, "Successfully pushed: 1") {
		t.Errorf("expected 'Successfully pushed: 1' in stdout, got: %s", stdout)
	}
}

func TestTranslationPush_MissingDir(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not be called when dir is missing")
	})

	stdout, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "translation", "push", "--workspace", "test-ws", "--dir", "/nonexistent/path/abc123")

	// translation push uses Run: not RunE: - exits 0 but prints nothing useful
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0 (Run: not RunE:); stdout: %s", exitCode, stdout)
	}
}

func TestTranslationPush_APIError(t *testing.T) {
	translationDir := setupTranslationDir(t)

	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"code": 401, "message": "Invalid service token"}`))
	})

	stdout, _, _ := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "bad-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "translation", "push", "--workspace", "test-ws", "--dir", translationDir)

	// translation push uses Run: not RunE: - error tracked in stats
	if !strings.Contains(stdout, "Failed to push: 1") {
		t.Errorf("expected 'Failed to push: 1' in stdout, got: %s", stdout)
	}
}
