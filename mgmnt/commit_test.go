package mgmnt

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/suprsend/cli/internal/clierr"
)

// The workflow and schema push endpoints answer 201/202 even when the commit
// fails, reporting the outcome in "commit_result". These bodies mirror the API.
func TestPushCommitResult(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		commit   bool
		wantCode string // "" means no error
	}{
		{"committed", `{"validation_result":{"is_valid":true},"commit_result":{"is_committed":true}}`, true, ""},
		{"nothing to commit", `{"validation_result":{"is_valid":true},"commit_result":{"is_committed":false,"errors":["no uncommitted changes detected"]}}`, true, ""},
		{"commit failed", `{"validation_result":{"is_valid":true},"commit_result":{"is_committed":false,"errors":["something broke"]}}`, true, clierr.CodeAPIConflict},
		{"no commit requested", `{"validation_result":{"is_valid":false,"errors":["bad"]}}`, false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()
			c := NewClientWithUrls("token", srv.URL, srv.URL, false)

			checkCode(t, "workflow", c.PushWorkflow("staging", "welcome", map[string]any{}, tt.commit, ""), tt.wantCode)
			checkCode(t, "schema", c.PushSchema("staging", "welcome", map[string]any{}, tt.commit, ""), tt.wantCode)
		})
	}
}

func TestPushWorkflowInvalidIsValidationError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"validation_result":{"is_valid":false,"errors":["node 1: template missing"]},"commit_result":{"is_committed":false,"errors":["error in workflow definition"]}}`))
	}))
	defer srv.Close()
	c := NewClientWithUrls("token", srv.URL, srv.URL, false)

	checkCode(t, "workflow", c.PushWorkflow("staging", "welcome", map[string]any{}, true, ""), clierr.CodeSchemaValidation)
}

func checkCode(t *testing.T, kind string, err error, wantCode string) {
	t.Helper()
	if wantCode == "" {
		if err != nil {
			t.Errorf("%s: unexpected error: %v", kind, err)
		}
		return
	}
	var ce *clierr.CLIError
	if !errors.As(err, &ce) {
		t.Fatalf("%s: got %v, want a clierr with code %s", kind, err, wantCode)
	}
	if ce.Code != wantCode {
		t.Errorf("%s: code = %s, want %s (%s)", kind, ce.Code, wantCode, ce.Message)
	}
}

// Commit endpoints answer 400/409 when the draft already matches live; a
// re-run of sync or commit must treat that as success.
func TestCommitNoChangesIsSuccess(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		wantCode string
	}{
		{"translation no changes", http.StatusBadRequest, `{"code":400,"type":"ValidationError","message":"['no changes to commit']"}`, ""},
		{"categories no changes", http.StatusConflict, `{"code":409,"message":"no changes to publish"}`, ""},
		{"template no changes", http.StatusConflict, `{"code":409,"error_code":"no_changes_to_commit","message":"no uncommitted changes detected in template"}`, ""},
		{"other 400", http.StatusBadRequest, `{"code":400,"message":"something else"}`, "any"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()
			c := NewClientWithUrls("token", srv.URL, srv.URL, false)

			calls := map[string]error{
				"workflow":    c.FinalizeWorkflow("staging", "welcome", ""),
				"schema":      c.FinalizeSchema("staging", "welcome", ""),
				"translation": c.FinalizeTranslation("staging", ""),
				"categories":  c.FinalizeCategories("staging", ""),
				"template":    c.CommitTemplate("staging", "welcome", "", nil),
			}
			for kind, err := range calls {
				if tt.wantCode == "" && err != nil {
					t.Errorf("%s: unexpected error: %v", kind, err)
				}
				if tt.wantCode != "" && err == nil {
					t.Errorf("%s: expected an error", kind)
				}
			}
		})
	}
}

func TestPushCategoriesCommitResult(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantCode string
	}{
		{"committed", `{"validation_result":{"is_valid":true},"commit_result":{"is_committed":true}}`, ""},
		{"nothing to publish", `{"validation_result":{"is_valid":true},"commit_result":{"is_committed":false,"errors":["no changes to publish"]}}`, ""},
		{"invalid", `{"validation_result":{"is_valid":false,"errors":["bad section"]},"commit_result":{"is_committed":false,"errors":["error in preference categories"]}}`, clierr.CodeSchemaValidation},
		{"commit failed", `{"validation_result":{"is_valid":true},"commit_result":{"is_committed":false,"errors":["db down"]}}`, clierr.CodeAPIConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()
			c := NewClientWithUrls("token", srv.URL, srv.URL, false)

			checkCode(t, "categories", c.PushCategories("staging", map[string]any{}, true, ""), tt.wantCode)
		})
	}
}
