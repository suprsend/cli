package mgmnt

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/suprsend/cli/internal/clierr"
)

// The workflow and schema push endpoints answer 201/202 even when the commit
// fails, reporting the outcome in "commit_result". These bodies mirror the API.
func TestPushCommitResult(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		commit     bool
		wantCode   string // "" means no error
		wantStatus CommitStatus
	}{
		{"committed", `{"validation_result":{"is_valid":true},"commit_result":{"is_committed":true}}`, true, "", Committed},
		{"nothing to commit", `{"validation_result":{"is_valid":true},"commit_result":{"is_committed":false,"errors":["no uncommitted changes detected"]}}`, true, "", NoChanges},
		{"commit failed", `{"validation_result":{"is_valid":true},"commit_result":{"is_committed":false,"errors":["something broke"]}}`, true, clierr.CodeAPIConflict, NotCommitted},
		{"no commit requested", `{"validation_result":{"is_valid":false,"errors":["bad"]}}`, false, "", NotCommitted},
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

			status, err := c.PushWorkflow("staging", "welcome", map[string]any{}, tt.commit, "")
			checkCode(t, "workflow", err, tt.wantCode)
			checkStatus(t, "workflow", status, tt.wantStatus)
			status, err = c.PushSchema("staging", "welcome", map[string]any{}, tt.commit, "")
			checkCode(t, "schema", err, tt.wantCode)
			checkStatus(t, "schema", status, tt.wantStatus)
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

	_, err := c.PushWorkflow("staging", "welcome", map[string]any{}, true, "")
	checkCode(t, "workflow", err, clierr.CodeSchemaValidation)
}

func checkStatus(t *testing.T, kind string, got, want CommitStatus) {
	t.Helper()
	if got != want {
		t.Errorf("%s: commit status = %q, want %q", kind, got, want)
	}
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

			type result struct {
				status CommitStatus
				err    error
			}
			do := func(s CommitStatus, err error) result { return result{s, err} }
			calls := map[string]result{
				"workflow":    do(c.FinalizeWorkflow("staging", "welcome", "")),
				"schema":      do(c.FinalizeSchema("staging", "welcome", "")),
				"translation": do(c.FinalizeTranslation("staging", "")),
				"categories":  do(c.FinalizeCategories("staging", "")),
				"template":    do(c.CommitTemplate("staging", "welcome", "", nil)),
			}
			for kind, r := range calls {
				if tt.wantCode == "" && r.err != nil {
					t.Errorf("%s: unexpected error: %v", kind, r.err)
				}
				if tt.wantCode == "" && r.status != NoChanges {
					t.Errorf("%s: commit status = %q, want %q", kind, r.status, NoChanges)
				}
				if tt.wantCode != "" && r.err == nil {
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

			_, err := c.PushCategories("staging", map[string]any{}, true, "")
			checkCode(t, "categories", err, tt.wantCode)
		})
	}
}

// A commit error that is only a quoted field name is the API's KeyError for a
// missing field; the message should say so instead of printing 'variable_ns'.
func TestCommitErrorDescribesMissingField(t *testing.T) {
	_, err := commitError("workflows/welcome", &CommitResult{Errors: []any{"'variable_ns'"}}, clierr.CodeAPIConflict)
	if err == nil || !strings.Contains(err.Error(), `required field "variable_ns" is missing`) {
		t.Errorf("got %v, want a message naming the missing field", err)
	}
}

// A 404 names the request, so a wrong workspace shows up in the message.
func TestNotFoundNamesRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	c := NewClientWithUrls("token", srv.URL, srv.URL, false)

	_, err := c.GetTemplate("evals-staging", "welcome", "live")
	checkCode(t, "template", err, clierr.CodeAPINotFound)
	if err == nil || !strings.Contains(err.Error(), "GET /v2/evals-staging/template/welcome/") {
		t.Errorf("got %v, want the request path in the message", err)
	}
}

// Gateway errors are retried, POST included, so a flaky gateway doesn't fail
// a push.
func TestRetriesGatewayErrors(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"validation_result":{"is_valid":true},"commit_result":{"is_committed":true}}`))
	}))
	defer srv.Close()
	c := NewClientWithUrls("token", srv.URL, srv.URL, false)

	status, err := c.PushWorkflow("staging", "welcome", map[string]any{}, true, "")
	if err != nil || status != Committed {
		t.Fatalf("got (%q, %v), want committed after a retry", status, err)
	}
	if n := calls.Load(); n != 2 {
		t.Errorf("server saw %d requests, want 2", n)
	}
}

// A dropped connection on a commit is retried; if the commit went through on
// the first attempt, the retry answers "no changes" and the command succeeds.
func TestCommitRetriedAfterConnectionReset(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			conn, _, err := w.(http.Hijacker).Hijack()
			if err == nil {
				conn.Close()
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"message":"no uncommitted changes detected in workflow"}`))
	}))
	defer srv.Close()
	c := NewClientWithUrls("token", srv.URL, srv.URL, false)

	status, err := c.FinalizeWorkflow("staging", "welcome", "")
	if err != nil || status != NoChanges {
		t.Fatalf("got (%q, %v), want unchanged after a retry", status, err)
	}
}

// When every attempt loses the connection, the commit's outcome is unknown,
// and the error says so.
func TestCommitOutcomeUnknownAfterRetries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _, err := w.(http.Hijacker).Hijack()
		if err == nil {
			conn.Close()
		}
	}))
	defer srv.Close()
	c := NewClientWithUrls("token", srv.URL, srv.URL, false)

	_, err := c.FinalizeWorkflow("staging", "welcome", "")
	if err == nil || !strings.Contains(err.Error(), "commit status unknown") {
		t.Errorf("got %v, want a commit-status-unknown error", err)
	}
}
