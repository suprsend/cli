package cli_test

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupSchemaDir creates a temp dir with a schema JSON file.
func setupSchemaDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	data := readTestdata(t, "schemas", "schema_pull_response.json")
	if err := os.WriteFile(filepath.Join(tmpDir, "test-schema.json"), data, 0644); err != nil {
		t.Fatalf("failed to write schema file: %v", err)
	}
	return tmpDir
}

func TestSchemaPush_CommitTrue(t *testing.T) {
	schemaDir := setupSchemaDir(t)

	var capturedMethod, capturedPath, capturedCommit string
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		capturedCommit = r.URL.Query().Get("commit")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	})

	stdout, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "schema", "push", "--workspace", "test-ws", "--dir", schemaDir, "--commit", "true")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout: %s", exitCode, stdout)
	}
	if capturedMethod != http.MethodPost {
		t.Errorf("expected POST, got %s", capturedMethod)
	}
	if !strings.Contains(capturedPath, "schema") {
		t.Errorf("unexpected path: %s", capturedPath)
	}
	if capturedCommit != "true" {
		t.Errorf("expected commit=true, got: %s", capturedCommit)
	}
	if !strings.Contains(stdout, "Successfully pushed: 1") {
		t.Errorf("expected 'Successfully pushed: 1' in stdout, got: %s", stdout)
	}
}

func TestSchemaPush_CommitFalse(t *testing.T) {
	schemaDir := setupSchemaDir(t)

	var capturedCommit string
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedCommit = r.URL.Query().Get("commit")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	})

	stdout, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "schema", "push", "--workspace", "test-ws", "--dir", schemaDir, "--commit", "false")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout: %s", exitCode, stdout)
	}
	if capturedCommit != "false" {
		t.Errorf("expected commit=false, got: %s", capturedCommit)
	}
}

func TestSchemaPush_MissingDir(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not be called when dir is missing")
	})

	_, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "schema", "push", "--workspace", "test-ws", "--dir", "/nonexistent/path/abc123")

	if exitCode == 0 {
		t.Error("expected non-zero exit code when dir is missing, got 0")
	}
}

func TestSchemaPush_APIError(t *testing.T) {
	schemaDir := setupSchemaDir(t)

	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"code": 401, "message": "Invalid service token"}`))
	})

	stdout, _, _ := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "bad-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "schema", "push", "--workspace", "test-ws", "--dir", schemaDir)

	// Push command reports failure in summary but exits 0 (errors are tracked in stats)
	if !strings.Contains(stdout, "Failed to push: 1") {
		t.Errorf("expected 'Failed to push: 1' in stdout, got: %s", stdout)
	}
}
