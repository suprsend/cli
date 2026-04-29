package category_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/suprsend/cli/tests/integration/helpers"
)

const defaultCategoryDir = "suprsend/preference_categories"

func newCategoryPullServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/staging/preference_category/", func(w http.ResponseWriter, r *http.Request) {
		serveFixture(t, w, "category", "pull_preference_category.json")
	})
	mux.HandleFunc("/v1/staging/preference_category/translation/locale", func(w http.ResponseWriter, r *http.Request) {
		serveFixture(t, w, "category", "pull_translation_locales.json")
	})
	for _, locale := range []string{"en", "es"} {
		mux.HandleFunc("/v1/staging/preference_category/translation/content/"+locale, func(w http.ResponseWriter, r *http.Request) {
			serveFixture(t, w, "category", "pull_translation_content_"+locale+".json")
		})
	}

	return httptest.NewServer(mux)
}

// TestCategoryPull_PromptNonInteractive verifies that when --dir is omitted and
// stdin is a pipe (non-interactive), the command exits 0 with a message instead
// of hanging on a prompt.
func TestCategoryPull_PromptNonInteractive(t *testing.T) {
	workDir := t.TempDir()

	// Pipe stdin so IsInputInteractive() returns false, triggering the non-interactive guard.
	out, err := helpers.RunCLIWithStdin(t, workDir,
		strings.NewReader(""),
		[]string{
			"SUPRSEND_SERVICE_TOKEN=test-token",
			"SUPRSEND_MGMNT_URL=http://127.0.0.1:0",
		},
		"category", "pull",
	)
	if err != nil {
		t.Fatalf("expected exit 0, got error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "cannot prompt in non-interactive mode") {
		t.Errorf("expected non-interactive message in output, got: %s", out)
	}

	// No files should have been written.
	if _, err := os.Stat(filepath.Join(workDir, defaultCategoryDir)); !os.IsNotExist(err) {
		t.Errorf("expected no output dir to be created, but it exists")
	}
}

// TestCategoryPull_PromptInteractive verifies that when --dir is omitted and stdin is
// a TTY, the prompt shows and accepts a directory path typed by the user.
func TestCategoryPull_PromptInteractive(t *testing.T) {
	srv := newCategoryPullServer(t)
	defer srv.Close()

	workDir := t.TempDir()
	customDir := filepath.Join(workDir, "my-categories")

	out, err := helpers.RunCLIWithPTY(t, workDir,
		customDir+"\n",
		[]string{
			"SUPRSEND_SERVICE_TOKEN=test-token",
			"SUPRSEND_MGMNT_URL=" + srv.URL,
		},
		"category", "pull",
	)
	if err != nil {
		t.Fatalf("expected exit 0, got error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "Where would you like to save the categories?") {
		t.Errorf("expected prompt text in output, got: %s", out)
	}
	if _, err := os.Stat(filepath.Join(customDir, "categories.json")); os.IsNotExist(err) {
		t.Errorf("expected categories.json to be written to %s\noutput: %s", customDir, out)
	}
}

// TestCategoryPull_PromptSkippedWhenDirExists verifies that when --dir is omitted
// but the default directory already exists, the prompt is skipped and files are written.
func TestCategoryPull_PromptSkippedWhenDirExists(t *testing.T) {
	srv := newCategoryPullServer(t)
	defer srv.Close()

	// Pre-create the default output directory so the prompt check is bypassed.
	workDir := t.TempDir()
	defaultDir := filepath.Join(workDir, "suprsend", "preference_categories")
	if err := os.MkdirAll(defaultDir, 0755); err != nil {
		t.Fatalf("failed to create default dir: %v", err)
	}

	_, err := helpers.RunCLIInDir(t, workDir,
		[]string{
			"SUPRSEND_SERVICE_TOKEN=test-token",
			"SUPRSEND_MGMNT_URL=" + srv.URL,
		},
		"category", "pull",
	)
	if err != nil {
		t.Fatalf("expected exit 0, got error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(defaultDir, "categories.json")); os.IsNotExist(err) {
		t.Errorf("expected categories.json to be written to default dir %s", defaultDir)
	}
}

func TestCategoryPull_ForceSkipsPrompt(t *testing.T) {
	srv := newCategoryPullServer(t)
	defer srv.Close()

	// use a fresh temp dir as working directory so ./suprsend/preference_categories doesn't exist
	workDir := t.TempDir()

	_, err := helpers.RunCLIInDir(t, workDir,
		[]string{
			"SUPRSEND_SERVICE_TOKEN=test-token",
			"SUPRSEND_MGMNT_URL=" + srv.URL,
		},
		"category", "pull", "--force",
	)
	if err != nil {
		t.Fatalf("expected exit 0, got error: %v", err)
	}

	defaultOut := filepath.Join(workDir, "suprsend", "preference_categories")
	if _, err := os.Stat(filepath.Join(defaultOut, "categories.json")); os.IsNotExist(err) {
		t.Errorf("expected categories.json at default path %s", defaultOut)
	}
	for _, locale := range []string{"en", "es"} {
		translationFile := filepath.Join(defaultOut, "translations", locale+".json")
		if _, err := os.Stat(translationFile); os.IsNotExist(err) {
			t.Errorf("expected translation file %s", translationFile)
		}
	}
}

func TestCategoryPull_DraftMode(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/staging/preference_category/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("mode") != "draft" {
			http.Error(w, `{"message":"expected mode=draft"}`, http.StatusBadRequest)
			return
		}
		serveFixture(t, w, "category", "pull_preference_category_draft.json")
	})
	mux.HandleFunc("/v1/staging/preference_category/translation/locale", func(w http.ResponseWriter, r *http.Request) {
		serveFixture(t, w, "category", "pull_translation_locales.json")
	})
	for _, locale := range []string{"en", "es"} {
		mux.HandleFunc("/v1/staging/preference_category/translation/content/"+locale, func(w http.ResponseWriter, r *http.Request) {
			serveFixture(t, w, "category", "pull_translation_content_"+locale+".json")
		})
	}
	srv := httptest.NewServer(mux)
	defer srv.Close()

	outDir := t.TempDir()

	_, err := helpers.RunCLI(t,
		[]string{
			"SUPRSEND_SERVICE_TOKEN=test-token",
			"SUPRSEND_MGMNT_URL=" + srv.URL,
		},
		"category", "pull", "--dir", outDir, "--mode", "draft",
	)
	if err != nil {
		t.Fatalf("expected exit 0, got error: %v", err)
	}
}

func TestCategoryPull_Success(t *testing.T) {
	srv := newCategoryPullServer(t)
	defer srv.Close()

	outDir := t.TempDir()

	_, err := helpers.RunCLI(t,
		[]string{
			"SUPRSEND_SERVICE_TOKEN=test-token",
			"SUPRSEND_MGMNT_URL=" + srv.URL,
		},
		"category", "pull", "--dir", outDir,
	)
	if err != nil {
		t.Fatalf("expected exit 0, got error: %v", err)
	}

	categoriesFile := filepath.Join(outDir, "categories.json")
	if _, err := os.Stat(categoriesFile); os.IsNotExist(err) {
		t.Errorf("expected categories.json to be written to %s", categoriesFile)
	}

	for _, locale := range []string{"en", "es"} {
		translationFile := filepath.Join(outDir, "translations", locale+".json")
		if _, err := os.Stat(translationFile); os.IsNotExist(err) {
			t.Errorf("expected translation file %s to be written", translationFile)
		}
	}
}
