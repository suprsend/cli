package category_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/suprsend/cli/tests/integration/helpers"
)

func newCategoryListServer(t *testing.T, mode string) *httptest.Server {
	t.Helper()
	fixture := "preference_category_list.json"
	if mode == "draft" {
		fixture = "preference_category_list_draft.json"
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/staging/preference_category/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("mode") != mode {
			http.Error(w, `{"message":"unexpected mode"}`, http.StatusBadRequest)
			return
		}
		serveFixture(t, w, "category", fixture)
	})
	return httptest.NewServer(mux)
}

func TestCategoryList_Success(t *testing.T) {
	srv := newCategoryListServer(t, "live")
	defer srv.Close()

	out, err := helpers.RunCLI(t,
		[]string{
			"SUPRSEND_SERVICE_TOKEN=test-token",
			"SUPRSEND_MGMNT_URL=" + srv.URL,
		},
		"category", "list",
	)
	if err != nil {
		t.Fatalf("expected exit 0, got error: %v\noutput: %s", err, out)
	}
}

func TestCategoryList_JSONOutput(t *testing.T) {
	srv := newCategoryListServer(t, "live")
	defer srv.Close()

	out, err := helpers.RunCLI(t,
		[]string{
			"SUPRSEND_SERVICE_TOKEN=test-token",
			"SUPRSEND_MGMNT_URL=" + srv.URL,
		},
		"category", "list", "--output", "json",
	)
	if err != nil {
		t.Fatalf("expected exit 0, got error: %v\noutput: %s", err, out)
	}

	var rows []map[string]any
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("output is not valid JSON array: %v\noutput: %s", err, out)
	}
	if len(rows) == 0 {
		t.Fatal("expected at least one category row in JSON output")
	}
	for _, key := range []string{"root_category", "section", "category_name", "default_preference"} {
		if _, ok := rows[0][key]; !ok {
			t.Errorf("expected key %q in first row, got: %v", key, rows[0])
		}
	}
}

func TestCategoryList_YAMLOutput(t *testing.T) {
	srv := newCategoryListServer(t, "live")
	defer srv.Close()

	out, err := helpers.RunCLI(t,
		[]string{
			"SUPRSEND_SERVICE_TOKEN=test-token",
			"SUPRSEND_MGMNT_URL=" + srv.URL,
		},
		"category", "list", "--output", "yaml",
	)
	if err != nil {
		t.Fatalf("expected exit 0, got error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "rootcategory:") {
		t.Errorf("expected YAML output to contain 'rootcategory:', got: %s", out)
	}
}

func TestCategoryList_DraftMode(t *testing.T) {
	srv := newCategoryListServer(t, "draft")
	defer srv.Close()

	out, err := helpers.RunCLI(t,
		[]string{
			"SUPRSEND_SERVICE_TOKEN=test-token",
			"SUPRSEND_MGMNT_URL=" + srv.URL,
		},
		"category", "list", "--mode", "draft",
	)
	if err != nil {
		t.Fatalf("expected exit 0, got error: %v\noutput: %s", err, out)
	}
}

func TestCategoryList_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"internal server error"}`, http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := helpers.RunCLI(t,
		[]string{
			"SUPRSEND_SERVICE_TOKEN=test-token",
			"SUPRSEND_MGMNT_URL=" + srv.URL,
		},
		"category", "list",
	)
	helpers.AssertExitCode(t, err, 1)
}
