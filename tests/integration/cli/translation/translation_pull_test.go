package cli_test

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestTranslationPull_CreatesFiles(t *testing.T) {
	pullData := readTestdata(t, "translations", "translation_pull_response.json")

	var reqCount int32
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&reqCount, 1)
		w.Header().Set("Content-Type", "application/json")
		if count == 1 {
			w.Write(pullData)
		} else {
			// Second page: empty results stops pagination
			w.Write([]byte(`{"meta":{"count":0,"limit":10,"offset":10},"results":[]}`))
		}
	})

	tmpDir := t.TempDir()
	stdout, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "translation", "pull", "--workspace", "test-ws", "--force", "--dir", tmpDir)

	// translation pull uses Run: not RunE: so exits 0 always
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout: %s", exitCode, stdout)
	}

	outFile := filepath.Join(tmpDir, "en.json")
	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("expected output file %s to exist: %v", outFile, err)
	}
	for _, want := range []string{"hello", "world", "greeting"} {
		if !strings.Contains(string(content), want) {
			t.Errorf("expected %q in output file, not found", want)
		}
	}
}

func TestTranslationPull_APIError(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"code": 401, "message": "Invalid service token"}`))
	})

	stdout, _, _ := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "bad-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "translation", "pull", "--workspace", "test-ws", "--force", "--dir", t.TempDir())

	// translation pull uses Run: not RunE: - error message appears in stdout
	if !strings.Contains(stdout, "Error") {
		t.Errorf("expected error message in stdout for API error, got: %s", stdout)
	}
}

func TestTranslationPull_IncludesContent(t *testing.T) {
	pullData := readTestdata(t, "translations", "translation_pull_response.json")

	var capturedQuery string
	var reqCount int32
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&reqCount, 1)
		capturedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		if count == 1 {
			w.Write(pullData)
		} else {
			w.Write([]byte(`{"results":[]}`))
		}
	})

	_, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "translation", "pull", "--workspace", "test-ws", "--force", "--dir", t.TempDir())

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
	if !strings.Contains(capturedQuery, "include_content=true") {
		t.Errorf("expected include_content=true in query, got: %s", capturedQuery)
	}
}
