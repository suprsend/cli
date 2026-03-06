package cli_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestCategoryList_Live(t *testing.T) {
	listData := readTestdata(t, "categories", "category_list_response.json")

	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Verify the request path and mode query param
		if !strings.Contains(r.URL.Path, "preference_category") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("mode") != "live" {
			t.Errorf("expected mode=live, got: %s", r.URL.Query().Get("mode"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(listData)
	})

	stdout, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "category", "list", "--workspace", "test-ws")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout: %s", exitCode, stdout)
	}

	// Verify root categories are present
	for _, rootCat := range []string{"system", "transactional", "promotional"} {
		if !strings.Contains(stdout, rootCat) {
			t.Errorf("expected root category %q in stdout, got: %s", rootCat, stdout)
		}
	}

	// Verify category names from test data
	for _, catName := range []string{"cat-system", "Sub Category", "return", "Refund promotion", "check bug1"} {
		if !strings.Contains(stdout, catName) {
			t.Errorf("expected category name %q in stdout, got: %s", catName, stdout)
		}
	}

	// Verify section names
	for _, section := range []string{"system section", "Nik Section", "Transactional karthick1"} {
		if !strings.Contains(stdout, section) {
			t.Errorf("expected section %q in stdout, got: %s", section, stdout)
		}
	}

	// Verify preference values
	for _, pref := range []string{"opt_in", "cant_unsubscribe"} {
		if !strings.Contains(stdout, pref) {
			t.Errorf("expected preference %q in stdout, got: %s", pref, stdout)
		}
	}
}

func TestCategoryList_InvalidMode(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Should not be called for invalid mode
		t.Error("server should not be called for invalid mode")
	})

	_, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "category", "list", "--workspace", "test-ws", "--mode", "invalid")

	if exitCode == 0 {
		t.Error("expected non-zero exit code for invalid mode, got 0")
	}
}

func TestCategoryList_APIError(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"code": 401, "message": "Invalid service token"}`))
	})

	_, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "bad-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "category", "list", "--workspace", "test-ws")

	if exitCode == 0 {
		t.Error("expected non-zero exit code for API error, got 0")
	}
}

func TestCategoryList_EmptyResponse(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"root_categories": [], "hash": "", "status": "live"}`))
	})

	stdout, _, exitCode := runCLIWithEnv(t, map[string]string{
		"SUPRSEND_SERVICE_TOKEN": "test-token",
		"SUPRSEND_MGMNT_URL":     server.URL,
	}, "category", "list", "--workspace", "test-ws")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout: %s", exitCode, stdout)
	}
}
