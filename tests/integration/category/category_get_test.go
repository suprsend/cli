package category_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/suprsend/cli/tests/integration/helpers"
)

var fixturesDir = helpers.TestdataPath

func serveFixture(t *testing.T, w http.ResponseWriter, parts ...string) {
	t.Helper()
	data, err := os.ReadFile(fixturesDir(t, parts...))
	if err != nil {
		http.Error(w, "fixture not found: "+filepath.Join(parts...), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func newCategoryGetServer(t *testing.T) *httptest.Server {
	t.Helper()
	base := []string{"category"}
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/staging/preference_category/", func(w http.ResponseWriter, r *http.Request) {
		serveFixture(t, w, append(base, "preference_category.json")...)
	})
	mux.HandleFunc("/v1/staging/preference_category/translation/locale", func(w http.ResponseWriter, r *http.Request) {
		serveFixture(t, w, append(base, "translation_locales.json")...)
	})
	for _, locale := range []string{"en-NA", "es", "es-AR", "es-BO"} {
		locale := locale
		filename := "translation_content_" + strings.ReplaceAll(locale, "-", "_") + ".json"
		mux.HandleFunc("/v1/staging/preference_category/translation/content/"+locale, func(w http.ResponseWriter, r *http.Request) {
			serveFixture(t, w, append(base, filename)...)
		})
	}

	return httptest.NewServer(mux)
}

func TestCategoryGet_Success(t *testing.T) {
	srv := newCategoryGetServer(t)
	defer srv.Close()

	out, err := helpers.RunCLI(t,
		[]string{
			"SUPRSEND_SERVICE_TOKEN=test-token",
			"SUPRSEND_MGMNT_URL=" + srv.URL,
		},
		"category", "get",
	)
	if err != nil {
		t.Fatalf("expected exit 0, got error: %v\noutput: %s", err, out)
	}
}

func TestCategoryGet_JSONOutput(t *testing.T) {
	srv := newCategoryGetServer(t)
	defer srv.Close()

	out, err := helpers.RunCLI(t,
		[]string{
			"SUPRSEND_SERVICE_TOKEN=test-token",
			"SUPRSEND_MGMNT_URL=" + srv.URL,
		},
		"category", "get",
	)
	if err != nil {
		t.Fatalf("expected exit 0, got error: %v\noutput: %s", err, out)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
	}
	if result["categories"] == nil {
		t.Errorf("expected 'categories' key in JSON output, got: %s", out)
	}
	if result["translations"] == nil {
		t.Errorf("expected 'translations' key in JSON output, got: %s", out)
	}
}

func TestCategoryGet_YAMLOutput(t *testing.T) {
	srv := newCategoryGetServer(t)
	defer srv.Close()

	out, err := helpers.RunCLI(t,
		[]string{
			"SUPRSEND_SERVICE_TOKEN=test-token",
			"SUPRSEND_MGMNT_URL=" + srv.URL,
		},
		"category", "get", "--output", "yaml",
	)
	if err != nil {
		t.Fatalf("expected exit 0, got error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "categories:") {
		t.Errorf("expected YAML output to contain 'categories:', got: %s", out)
	}
	if !strings.Contains(out, "translations:") {
		t.Errorf("expected YAML output to contain 'translations:', got: %s", out)
	}
}

func TestCategoryGet_InvalidOutputFormat(t *testing.T) {
	srv := newCategoryGetServer(t)
	defer srv.Close()

	_, err := helpers.RunCLI(t,
		[]string{
			"SUPRSEND_SERVICE_TOKEN=test-token",
			"SUPRSEND_MGMNT_URL=" + srv.URL,
		},
		"category", "get", "--output", "table",
	)
	if err == nil {
		t.Fatal("expected non-zero exit for invalid output format, but got exit 0")
	}
}

func TestCategoryGet_DraftMode(t *testing.T) {
	base := []string{"category"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/staging/preference_category/":
			if r.URL.Query().Get("mode") != "draft" {
				http.Error(w, `{"message":"expected mode=draft"}`, http.StatusBadRequest)
				return
			}
			serveFixture(t, w, append(base, "preference_category_draft.json")...)
		case "/v1/staging/preference_category/translation/locale":
			serveFixture(t, w, append(base, "translation_locales.json")...)
		case "/v1/staging/preference_category/translation/content/en-NA":
			serveFixture(t, w, append(base, "translation_content_en_NA.json")...)
		case "/v1/staging/preference_category/translation/content/es":
			serveFixture(t, w, append(base, "translation_content_es.json")...)
		case "/v1/staging/preference_category/translation/content/es-AR":
			serveFixture(t, w, append(base, "translation_content_es_AR.json")...)
		case "/v1/staging/preference_category/translation/content/es-BO":
			serveFixture(t, w, append(base, "translation_content_es_BO.json")...)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	out, err := helpers.RunCLI(t,
		[]string{
			"SUPRSEND_SERVICE_TOKEN=test-token",
			"SUPRSEND_MGMNT_URL=" + srv.URL,
		},
		"category", "get", "--mode", "draft",
	)
	if err != nil {
		t.Fatalf("expected exit 0, got error: %v\noutput: %s", err, out)
	}
}
