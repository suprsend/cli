package translation

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/cli/mgmnt"
)

// runTranslationPull points the management client at handler and runs
// `translation pull` with args.
func runTranslationPull(t *testing.T, handler http.HandlerFunc, args ...string) error {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	prevClient := utils.SDKInstance
	utils.SDKInstance = mgmnt.NewClientWithUrls("test-token", srv.URL, srv.URL, false)
	t.Cleanup(func() { utils.SDKInstance = prevClient })

	prevSilenceErrors, prevSilenceUsage := TranslationCmd.SilenceErrors, TranslationCmd.SilenceUsage
	TranslationCmd.SilenceErrors, TranslationCmd.SilenceUsage = true, true
	t.Cleanup(func() {
		TranslationCmd.SilenceErrors, TranslationCmd.SilenceUsage = prevSilenceErrors, prevSilenceUsage
		TranslationCmd.SetArgs(nil)
		// cobra keeps parsed flag values between Execute calls; reset them so
		// one test's flags don't leak into the next.
		resetFlags := func(f *pflag.Flag) {
			_ = f.Value.Set(f.DefValue)
			f.Changed = false
		}
		translationPullCmd.Flags().VisitAll(resetFlags)
		TranslationCmd.PersistentFlags().VisitAll(resetFlags)
	})

	TranslationCmd.SetArgs(append([]string{"pull"}, args...))
	return TranslationCmd.Execute()
}

func TestTranslationPullReturnsAPIError(t *testing.T) {
	dir := t.TempDir()
	err := runTranslationPull(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"boom"}`))
	}, "--dir", dir, "--workspace", "staging")

	if err == nil {
		t.Fatal("expected translation pull to return an error when the API fails")
	}
	var ce *clierr.CLIError
	if !errors.As(err, &ce) || ce.Code != clierr.CodeAPIInternal {
		t.Errorf("expected a clierr with code %q, got %#v", clierr.CodeAPIInternal, err)
	}
}

func TestTranslationPullRejectsInvalidMode(t *testing.T) {
	err := runTranslationPull(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected API call: %s %s", r.Method, r.URL)
	}, "--dir", t.TempDir(), "--mode", "published")

	var ce *clierr.CLIError
	if !errors.As(err, &ce) || ce.Code != clierr.CodeInvalidUsage {
		t.Errorf("expected a clierr with code %q, got %#v", clierr.CodeInvalidUsage, err)
	}
}

func TestTranslationPullWritesFiles(t *testing.T) {
	dir := t.TempDir()
	err := runTranslationPull(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("offset") == "0" {
			_, _ = w.Write([]byte(`{"results":[{"slug":"candidate","filename":"candidate.es.json","content":{"hello":"hola"}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"results":[]}`))
	}, "--dir", dir)
	if err != nil {
		t.Fatalf("translation pull returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "candidate.es.json"))
	if err != nil {
		t.Fatalf("expected candidate.es.json to be written: %v", err)
	}
	if want := "{\n  \"hello\": \"hola\"\n}"; string(data) != want {
		t.Errorf("candidate.es.json = %q, want %q", data, want)
	}
}
