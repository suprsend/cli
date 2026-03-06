package cli_test

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
)

func TestEventList_Success(t *testing.T) {
	listData := readTestdata(t, "events", "event_list_response.json")

	var capturedMethod, capturedPath string
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write(listData)
	})

	stdout, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "event", "list", "--workspace", "test-ws")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout: %s", exitCode, stdout)
	}
	if capturedMethod != http.MethodGet {
		t.Errorf("expected GET, got %s", capturedMethod)
	}
	if !strings.Contains(capturedPath, "event") {
		t.Errorf("unexpected path: %s", capturedPath)
	}

	// Verify event names appear in output
	for _, name := range []string{"ORDER_CREATED", "SANJEEV EVENT", "SREEHARI_TEST"} {
		if !strings.Contains(stdout, name) {
			t.Errorf("expected event %q in stdout, got: %s", name, stdout)
		}
	}
}

func TestEventList_DefaultLimitOffset(t *testing.T) {
	listData := readTestdata(t, "events", "event_list_response.json")

	var capturedLimit, capturedOffset string
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedLimit = r.URL.Query().Get("limit")
		capturedOffset = r.URL.Query().Get("offset")
		w.Header().Set("Content-Type", "application/json")
		w.Write(listData)
	})

	_, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "event", "list", "--workspace", "test-ws")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
	if capturedLimit != "20" {
		t.Errorf("expected default limit=20, got: %s", capturedLimit)
	}
	if capturedOffset != "0" {
		t.Errorf("expected default offset=0, got: %s", capturedOffset)
	}
}

func TestEventList_WithLimitOffset(t *testing.T) {
	listData := readTestdata(t, "events", "event_list_response.json")

	var capturedLimit, capturedOffset string
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedLimit = r.URL.Query().Get("limit")
		capturedOffset = r.URL.Query().Get("offset")
		w.Header().Set("Content-Type", "application/json")
		w.Write(listData)
	})

	_, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "event", "list", "--workspace", "test-ws", "--limit", "5", "--offset", "10")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
	if capturedLimit != strconv.Itoa(5) {
		t.Errorf("expected limit=5, got: %s", capturedLimit)
	}
	if capturedOffset != strconv.Itoa(10) {
		t.Errorf("expected offset=10, got: %s", capturedOffset)
	}
}

func TestEventList_EmptyResults(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"meta":{"count":0,"limit":20,"offset":0},"results":[]}`))
	})

	stdout, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "event", "list", "--workspace", "test-ws")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout: %s", exitCode, stdout)
	}
}

func TestEventList_APIError(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"code": 401, "message": "Invalid service token"}`))
	})

	_, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "bad-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "event", "list", "--workspace", "test-ws")

	if exitCode == 0 {
		t.Error("expected non-zero exit code for API error, got 0")
	}
}
