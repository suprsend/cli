package cli_test

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestEventPull_CreatesFile(t *testing.T) {
	pullData := readTestdata(t, "events", "event_pull_response.json")

	var reqCount int32
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&reqCount, 1)
		w.Header().Set("Content-Type", "application/json")
		if count == 1 {
			w.Write(pullData)
		} else {
			// Second page: empty results stops pagination
			w.Write([]byte(`{"meta":{"count":0,"limit":50,"offset":50},"results":[]}`))
		}
	})

	tmpDir := t.TempDir()
	stdout, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "event", "pull", "--workspace", "test-ws", "--force", "--dir", tmpDir)

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout: %s", exitCode, stdout)
	}

	outFile := filepath.Join(tmpDir, "event_schema_mapping.json")
	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("expected output file %s to exist: %v", outFile, err)
	}

	for _, want := range []string{"events", "ORDER_CREATED", "PAYMENT_DONE"} {
		if !strings.Contains(string(content), want) {
			t.Errorf("expected %q in output file, not found", want)
		}
	}
}

func TestEventPull_APIError(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"code": 401, "message": "Invalid service token"}`))
	})

	_, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "bad-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "event", "pull", "--workspace", "test-ws", "--force", "--dir", t.TempDir())

	if exitCode == 0 {
		t.Error("expected non-zero exit code for API error, got 0")
	}
}

func TestEventPull_HasLinkedSchemaParam(t *testing.T) {
	pullData := readTestdata(t, "events", "event_pull_response.json")

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
	}, "event", "pull", "--workspace", "test-ws", "--force", "--dir", t.TempDir())

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
	if !strings.Contains(capturedQuery, "has_linked_schema=true") {
		t.Errorf("expected has_linked_schema=true in query, got: %s", capturedQuery)
	}
}
