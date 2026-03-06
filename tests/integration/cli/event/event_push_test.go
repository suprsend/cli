package cli_test

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupEventDir creates a temp dir with event_schema_mapping.json populated from testdata.
func setupEventDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	data := readTestdata(t, "events", "event_schema_mapping.json")
	if err := os.WriteFile(filepath.Join(tmpDir, "event_schema_mapping.json"), data, 0644); err != nil {
		t.Fatalf("failed to write event schema mapping file: %v", err)
	}
	return tmpDir
}

func TestEventPush_Success(t *testing.T) {
	eventDir := setupEventDir(t)

	var capturedMethod, capturedPath string
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	})

	stdout, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "event", "push", "--workspace", "test-ws", "--dir", eventDir)

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout: %s", exitCode, stdout)
	}
	if capturedMethod != http.MethodPost {
		t.Errorf("expected POST, got %s", capturedMethod)
	}
	if !strings.Contains(capturedPath, "bulk") || !strings.Contains(capturedPath, "event") {
		t.Errorf("unexpected path: %s", capturedPath)
	}
}

func TestEventPush_MissingFile(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not be called when input file is missing")
	})

	emptyDir := t.TempDir()

	_, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "event", "push", "--workspace", "test-ws", "--dir", emptyDir)

	if exitCode == 0 {
		t.Error("expected non-zero exit code when input file is missing, got 0")
	}
}

func TestEventPush_APIError(t *testing.T) {
	eventDir := setupEventDir(t)

	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"code": 401, "message": "Invalid service token"}`))
	})

	_, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "bad-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "event", "push", "--workspace", "test-ws", "--dir", eventDir)

	if exitCode == 0 {
		t.Error("expected non-zero exit code for API error, got 0")
	}
}
