package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/state"
	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/cli/mgmnt"
)

// fakeAPI keeps workflow drafts by slug and counts pushes. Each save bumps
// updated_at and hash, as the real API does, so tests see server-managed
// fields change without the definition changing.
type fakeAPI struct {
	mu     sync.Mutex
	drafts map[string]map[string]any
	pushes int
	saves  int
}

func (f *fakeAPI) handler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/") // v1/<ws>/workflow/<slug>[/commit]
		if len(parts) < 4 {
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		slug := parts[3]
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			wf, ok := f.drafts[slug]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"code":404,"message":"workflow not found"}`))
				return
			}
			_ = json.NewEncoder(w).Encode(wf)
		case http.MethodPost:
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			f.pushes++
			f.save(slug, body)
			_, _ = w.Write([]byte(`{"validation_result":{"is_valid":true},"commit_result":{"is_committed":true}}`))
		case http.MethodPatch: // commit
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"message":"no changes to commit"}`))
		}
	}
}

// save stores wf as the draft, with fresh server-managed fields.
func (f *fakeAPI) save(slug string, wf map[string]any) {
	f.saves++
	stored := map[string]any{}
	for k, v := range wf {
		stored[k] = v
	}
	stored["updated_at"] = fmt.Sprintf("t%d", f.saves)
	stored["hash"] = fmt.Sprintf("h%d", f.saves)
	stored["status"] = "draft"
	f.drafts[slug] = stored
}

func (f *fakeAPI) edit(slug, key string, value any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	wf := map[string]any{}
	for k, v := range f.drafts[slug] {
		wf[k] = v
	}
	wf[key] = value
	f.save(slug, wf)
}

func (f *fakeAPI) pushCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.pushes
}

func copyMap(m map[string]any) map[string]any {
	data, _ := json.Marshal(m)
	var out map[string]any
	_ = json.Unmarshal(data, &out)
	return out
}

func TestPushOneUsesState(t *testing.T) {
	api := &fakeAPI{drafts: map[string]map[string]any{}}
	api.save("welcome", map[string]any{"slug": "welcome", "name": "Welcome", "tree": map[string]any{"nodes": []any{"email"}}})
	srv := httptest.NewServer(api.handler(t))
	defer srv.Close()
	client := mgmnt.NewClientWithUrls("token", srv.URL+"/", srv.URL+"/", false)

	st, err := state.Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	// Pull: the local file is the workspace's draft, plus $schema.
	local := copyMap(api.drafts["welcome"])
	local["$schema"] = "https://schema.suprsend.com/workflow/v1/schema.json"
	RecordPulled(st, "staging", local)
	opts := pushOptions{state: st}

	push := func(wf map[string]any, o pushOptions) (string, error) {
		t.Helper()
		status, _, err := pushOne(client, "staging", "welcome", wf, o)
		return status, err
	}
	wantStatus := func(step, got string, err error, want string) {
		t.Helper()
		if err != nil || got != want {
			t.Fatalf("%s: got %q, %v; want %q", step, got, err, want)
		}
	}

	status, err := push(local, opts)
	wantStatus("push right after pull", status, err, utils.StatusUnchanged)
	if api.pushCount() != 0 {
		t.Fatal("an unchanged workflow was sent")
	}

	local["name"] = "Welcome!"
	status, err = push(local, opts)
	wantStatus("push a local edit", status, err, utils.StatusUpdated)

	status, err = push(local, opts)
	wantStatus("push it again", status, err, utils.StatusUnchanged)

	// Someone edits the draft in the dashboard; the local copy is now stale.
	api.edit("welcome", "tree", map[string]any{"nodes": []any{"email", "sms"}})
	before := api.pushCount()
	status, err = push(local, opts)
	wantStatus("push a stale copy", status, err, utils.StatusSkipped)
	if api.pushCount() != before {
		t.Fatal("a stale copy was sent over the dashboard edit")
	}

	_, _, err = pushOne(client, "staging", "welcome", local, pushOptions{state: st, commit: true})
	if err == nil {
		t.Fatal("committing over an unseen dashboard edit should fail")
	}

	local["name"] = "Welcome again"
	_, err = push(local, opts)
	var ce *clierr.CLIError
	if !errors.As(err, &ce) || ce.Code != clierr.CodeAPIConflict {
		t.Fatalf("edit on both sides: got %v, want a conflict error", err)
	}
	if api.pushCount() != before {
		t.Fatal("a conflicting copy was sent")
	}

	status, err = push(local, pushOptions{state: st, overwrite: true})
	wantStatus("--overwrite", status, err, utils.StatusUpdated)
	if got := api.drafts["welcome"]["name"]; got != "Welcome again" {
		t.Fatalf("draft name = %v after --overwrite", got)
	}

	// A workflow the workspace doesn't have is created, and recorded.
	newWF := map[string]any{"slug": "fresh", "name": "Fresh"}
	status, _, err = pushOne(client, "staging", "fresh", newWF, opts)
	wantStatus("new workflow", status, err, utils.StatusCreated)
	if _, ok := st.Get("staging", "fresh"); !ok {
		t.Fatal("a created workflow wasn't recorded")
	}

	// Files pulled from staging and pushed to production compare with
	// production alone: no record there, so no conflict.
	status, _, err = pushOne(client, "production", "welcome", local, opts)
	wantStatus("another workspace", status, err, utils.StatusUnchanged)
}
