package cli_test

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestSchemaPull_BySlug(t *testing.T) {
	pullData := readTestdata(t, "schemas", "schema_pull_response.json")

	var capturedPath, capturedMode string
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedMode = r.URL.Query().Get("mode")
		w.Header().Set("Content-Type", "application/json")
		w.Write(pullData)
	})

	tmpDir := t.TempDir()
	stdout, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "schema", "pull", "--workspace", "test-ws", "--slug", "test-schema", "--dir", tmpDir, "--force")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout: %s", exitCode, stdout)
	}
	if !strings.Contains(capturedPath, "test-schema") {
		t.Errorf("expected slug in path, got: %s", capturedPath)
	}
	if capturedMode != "live" {
		t.Errorf("expected default mode=live, got: %s", capturedMode)
	}

	outFile := filepath.Join(tmpDir, "test-schema.json")
	if _, err := os.Stat(outFile); err != nil {
		t.Errorf("expected output file %s to exist: %v", outFile, err)
	}
}

func TestSchemaPull_All(t *testing.T) {
	pullData := readTestdata(t, "schemas", "schemas_pull_all_response.json")

	var reqCount int32
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&reqCount, 1)
		w.Header().Set("Content-Type", "application/json")
		if count == 1 {
			w.Write(pullData)
		} else {
			w.Write([]byte(`{"meta":{"count":0,"limit":50,"offset":50},"results":[]}`))
		}
	})

	tmpDir := t.TempDir()
	stdout, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "schema", "pull", "--workspace", "test-ws", "--dir", tmpDir, "--force")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout: %s", exitCode, stdout)
	}

	for _, slug := range []string{"schema-alpha", "schema-beta"} {
		outFile := filepath.Join(tmpDir, slug+".json")
		if _, err := os.Stat(outFile); err != nil {
			t.Errorf("expected output file %s to exist: %v", outFile, err)
		}
	}
	if !strings.Contains(stdout, "Successfully updated: 2") {
		t.Errorf("expected 'Successfully updated: 2' in stdout, got: %s", stdout)
	}
}

func TestSchemaPull_APIError(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"code": 401, "message": "Invalid service token"}`))
	})

	_, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "bad-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "schema", "pull", "--workspace", "test-ws", "--slug", "test-schema", "--dir", t.TempDir(), "--force")

	if exitCode == 0 {
		t.Error("expected non-zero exit code for API error, got 0")
	}
}
