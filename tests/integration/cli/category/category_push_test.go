package cli_test

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupCategoryDir creates a temp dir with categories_preferences.json populated from testdata.
func setupCategoryDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	data := readTestdata(t, "categories", "categories_preferences.json")
	if err := os.WriteFile(filepath.Join(tmpDir, "categories_preferences.json"), data, 0644); err != nil {
		t.Fatalf("failed to write categories file: %v", err)
	}
	return tmpDir
}

// TestCategoryPush_CommitTrue verifies that push with commit=true sends a POST and succeeds.
func TestCategoryPush_CommitTrue(t *testing.T) {
	responseData := readTestdata(t, "categories", "category_push_response_commit.json")
	categoryDir := setupCategoryDir(t)

	var capturedMethod, capturedPath, capturedCommit string
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		capturedCommit = r.URL.Query().Get("commit")
		w.Header().Set("Content-Type", "application/json")
		w.Write(responseData)
	})

	stdout, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "category", "push", "--workspace", "test-ws", "--dir", categoryDir, "--commit", "true")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout: %s", exitCode, stdout)
	}
	if capturedMethod != http.MethodPost {
		t.Errorf("expected POST, got %s", capturedMethod)
	}
	if !strings.Contains(capturedPath, "preference_category") {
		t.Errorf("unexpected path: %s", capturedPath)
	}
	if capturedCommit != "true" {
		t.Errorf("expected commit=true, got: %s", capturedCommit)
	}
}

// TestCategoryPush_CommitFalse verifies that push with commit=false sends a POST with commit=false.
func TestCategoryPush_CommitFalse(t *testing.T) {
	responseData := readTestdata(t, "categories", "category_push_response_no_commit.json")
	categoryDir := setupCategoryDir(t)

	var capturedCommit string
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedCommit = r.URL.Query().Get("commit")
		w.Header().Set("Content-Type", "application/json")
		w.Write(responseData)
	})

	stdout, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "category", "push", "--workspace", "test-ws", "--dir", categoryDir, "--commit", "false")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout: %s", exitCode, stdout)
	}
	if capturedCommit != "false" {
		t.Errorf("expected commit=false, got: %s", capturedCommit)
	}
}

// TestCategoryPush_MissingFile verifies non-zero exit when categories_preferences.json is absent.
func TestCategoryPush_MissingFile(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not be called when input file is missing")
	})

	emptyDir := t.TempDir() // no categories_preferences.json inside

	_, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "category", "push", "--workspace", "test-ws", "--dir", emptyDir)

	if exitCode == 0 {
		t.Error("expected non-zero exit code when input file is missing, got 0")
	}
}

// TestCategoryPush_APIError verifies non-zero exit on API error response.
func TestCategoryPush_APIError(t *testing.T) {
	categoryDir := setupCategoryDir(t)

	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"code": 401, "message": "Invalid service token"}`))
	})

	_, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "bad-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "category", "push", "--workspace", "test-ws", "--dir", categoryDir, "--commit", "false")

	if exitCode == 0 {
		t.Error("expected non-zero exit code for API error, got 0")
	}
}

// TestCategoryPush_ValidationFailure verifies that a validation failure is reported but exits 0.
func TestCategoryPush_ValidationFailure(t *testing.T) {
	categoryDir := setupCategoryDir(t)

	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"validation_result":{"is_valid":false,"errors":["category slug must be unique"]}}`))
	})

	stdout, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "category", "push", "--workspace", "test-ws", "--dir", categoryDir, "--commit", "true")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout: %s", exitCode, stdout)
	}
	if !strings.Contains(stdout, "Warning") {
		t.Errorf("expected validation warning in stdout, got: %s", stdout)
	}
}
