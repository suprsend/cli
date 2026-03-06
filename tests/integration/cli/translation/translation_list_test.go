package cli_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestTranslationList_Success(t *testing.T) {
	listData := readTestdata(t, "translations", "translation_list_response.json")

	var capturedMethod string
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		w.Write(listData)
	})

	stdout, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "translation", "list", "--workspace", "test-ws")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout: %s", exitCode, stdout)
	}
	if capturedMethod != http.MethodGet {
		t.Errorf("expected GET, got %s", capturedMethod)
	}
	for _, locale := range []string{"en.json", "fr.json"} {
		if !strings.Contains(stdout, locale) {
			t.Errorf("expected locale %q in stdout, got: %s", locale, stdout)
		}
	}
}

func TestTranslationList_DefaultLimitOffset(t *testing.T) {
	listData := readTestdata(t, "translations", "translation_list_response.json")

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
	}, "translation", "list", "--workspace", "test-ws")

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

func TestTranslationList_EmptyResults(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"meta":{"count":0,"limit":20,"offset":0},"results":[]}`))
	})

	stdout, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "translation", "list", "--workspace", "test-ws")

	// translation list uses Run: not RunE: so exits 0 always
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout: %s", exitCode, stdout)
	}
}
