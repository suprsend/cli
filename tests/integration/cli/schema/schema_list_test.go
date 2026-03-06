package cli_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestSchemaList_Success(t *testing.T) {
	listData := readTestdata(t, "schemas", "schema_list_response.json")

	var capturedMethod string
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		w.Write(listData)
	})

	stdout, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "schema", "list", "--workspace", "test-ws")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout: %s", exitCode, stdout)
	}
	if capturedMethod != http.MethodGet {
		t.Errorf("expected GET, got %s", capturedMethod)
	}
	for _, slug := range []string{"schema-one", "schema-two"} {
		if !strings.Contains(stdout, slug) {
			t.Errorf("expected slug %q in stdout, got: %s", slug, stdout)
		}
	}
}

func TestSchemaList_DefaultLimitOffsetMode(t *testing.T) {
	listData := readTestdata(t, "schemas", "schema_list_response.json")

	var capturedLimit, capturedOffset, capturedMode string
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedLimit = r.URL.Query().Get("limit")
		capturedOffset = r.URL.Query().Get("offset")
		capturedMode = r.URL.Query().Get("mode")
		w.Header().Set("Content-Type", "application/json")
		w.Write(listData)
	})

	_, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "schema", "list", "--workspace", "test-ws")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
	if capturedLimit != "20" {
		t.Errorf("expected default limit=20, got: %s", capturedLimit)
	}
	if capturedOffset != "0" {
		t.Errorf("expected default offset=0, got: %s", capturedOffset)
	}
	if capturedMode != "live" {
		t.Errorf("expected default mode=live, got: %s", capturedMode)
	}
}

func TestSchemaList_DraftMode(t *testing.T) {
	listData := readTestdata(t, "schemas", "schema_list_response.json")

	var capturedMode string
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedMode = r.URL.Query().Get("mode")
		w.Header().Set("Content-Type", "application/json")
		w.Write(listData)
	})

	_, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "schema", "list", "--workspace", "test-ws", "--mode", "draft")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
	if capturedMode != "draft" {
		t.Errorf("expected mode=draft, got: %s", capturedMode)
	}
}

func TestSchemaList_APIError(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"code": 401, "message": "Invalid service token"}`))
	})

	_, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "bad-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "schema", "list", "--workspace", "test-ws")

	if exitCode == 0 {
		t.Error("expected non-zero exit code for API error, got 0")
	}
}
