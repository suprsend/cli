package cli_test

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCategoryPull_CreatesFile(t *testing.T) {
	pullData := readTestdata(t, "categories", "category_pull_response.json")

	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "translation/locale") {
			w.Write([]byte(`{"results":[]}`))
		} else {
			w.Write(pullData)
		}
	})

	tmpDir := t.TempDir()
	_, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "category", "pull", "--workspace", "test-ws", "--force", "--dir", tmpDir)

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}

	outFile := filepath.Join(tmpDir, "categories_preferences.json")
	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("expected output file %s to exist: %v", outFile, err)
	}

	for _, want := range []string{"root_categories", "akhil-system", "return"} {
		if !strings.Contains(string(content), want) {
			t.Errorf("expected %q in output file, not found", want)
		}
	}
}

func TestCategoryPull_InvalidMode(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not be called for invalid mode")
	})

	_, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "category", "pull", "--workspace", "test-ws", "--mode", "invalid", "--force", "--dir", t.TempDir())

	if exitCode == 0 {
		t.Error("expected non-zero exit code for invalid mode, got 0")
	}
}

func TestCategoryPull_APIError(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "translation/locale") {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"results":[]}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"code": 401, "message": "Invalid service token"}`))
	})

	_, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "bad-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "category", "pull", "--workspace", "test-ws", "--force", "--dir", t.TempDir())

	if exitCode == 0 {
		t.Error("expected non-zero exit code for API error, got 0")
	}
}
