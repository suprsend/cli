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
	// lastBody is the body of the latest push, as sent.
	lastBody map[string]any
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
			f.lastBody = body
			// Like the API: a null is_enabled keeps the stored value, and a
			// new workflow starts enabled.
			if v, ok := body["is_enabled"]; ok && v == nil {
				stored := any(true)
				if old, ok := f.drafts[slug]; ok && old["is_enabled"] != nil {
					stored = old["is_enabled"]
				}
				body = copyMap(body)
				body["is_enabled"] = stored
			}
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

	st, err := state.Load(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	// Pull: the local file is the workspace's draft, plus $schema.
	local := copyMap(api.drafts["welcome"])
	local["$schema"] = "https://schema.suprsend.com/workflow/v1/schema.json"
	RecordPulled(st, "staging", local, nil)
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

func multiChannel(name string, channels ...any) map[string]any {
	return map[string]any{"slug": "kyc", "name": name, "tree": map[string]any{"nodes": []any{
		map[string]any{"node_type": "send_multi_channel", "properties": map[string]any{"channels": channels, "template": "kyc"}},
	}}}
}

// #34: pull writes the live version, push compares with the draft.
func TestPushAfterLivePull(t *testing.T) {
	newClient := func(t *testing.T, draft map[string]any) (*fakeAPI, *mgmnt.SS_MgmntClient) {
		api := &fakeAPI{drafts: map[string]map[string]any{}}
		api.save("kyc", draft)
		srv := httptest.NewServer(api.handler(t))
		t.Cleanup(srv.Close)
		return api, mgmnt.NewClientWithUrls("token", srv.URL+"/", srv.URL+"/", false)
	}
	newState := func(t *testing.T) *state.File {
		st, err := state.Load(t.TempDir(), "")
		if err != nil {
			t.Fatal(err)
		}
		return st
	}

	t.Run("channel order is not a change", func(t *testing.T) {
		live := multiChannel("KYC", "sms", "email")
		api, client := newClient(t, multiChannel("KYC", "email", "sms"))
		st := newState(t)
		RecordPulled(st, "staging", live, copyMap(api.drafts["kyc"]))
		if e, _ := st.Get("staging", "kyc"); e.Pending {
			t.Fatal("a draft that differs only in channel order was recorded as pending")
		}
		status, _, err := pushOne(client, "staging", "kyc", live, pushOptions{state: st, commit: true})
		if err != nil || status != utils.StatusUnchanged || api.pushCount() != 0 {
			t.Fatalf("push --commit of an untouched workflow: %q, %v, %d pushes", status, err, api.pushCount())
		}
	})

	t.Run("a pending draft is left alone", func(t *testing.T) {
		live := multiChannel("KYC", "sms", "email")
		api, client := newClient(t, multiChannel("KYC, reworded", "sms", "email"))
		st := newState(t)
		RecordPulled(st, "staging", live, copyMap(api.drafts["kyc"]))

		for _, commit := range []bool{false, true} {
			status, detail, err := pushOne(client, "staging", "kyc", live, pushOptions{state: st, commit: commit})
			if err != nil || status != utils.StatusUnchanged || !strings.Contains(detail, "--mode draft") {
				t.Fatalf("commit=%v: got %q (%s), %v; want unchanged with a note", commit, status, detail, err)
			}
		}
		if api.pushCount() != 0 || api.drafts["kyc"]["name"] != "KYC, reworded" {
			t.Fatal("the pending draft was replaced")
		}

		// An edit based on live would replace the draft nobody here has seen.
		edited := multiChannel("KYC v2", "sms", "email")
		_, _, err := pushOne(client, "staging", "kyc", edited, pushOptions{state: st})
		if !clierr.HasCode(err, clierr.CodeAPIConflict) || api.pushCount() != 0 {
			t.Fatalf("edit over a pending draft: got %v, want a conflict and no push", err)
		}
		status, _, err := pushOne(client, "staging", "kyc", edited, pushOptions{state: st, overwrite: true})
		if err != nil || status != utils.StatusUpdated {
			t.Fatalf("--overwrite: %q, %v", status, err)
		}
		if e, _ := st.Get("staging", "kyc"); e.Pending {
			t.Fatal("the entry is still pending after the push")
		}
	})
}

// The API keeps these lists as sets and returns them in varying order.
func TestFingerprintIgnoresSetOrder(t *testing.T) {
	wf := func(events []any, mandatory []any) map[string]any {
		return map[string]any{"slug": "s", "trigger_events": events, "tree": map[string]any{"nodes": []any{
			map[string]any{"node_type": "send_smart_channel_routing", "properties": map[string]any{"mandatory_channels": mandatory, "template": "t"}},
		}}}
	}
	a := wf([]any{"A", "B"}, []any{"email", "sms"})
	if fingerprint(a) != fingerprint(wf([]any{"B", "A"}, []any{"sms", "email"})) {
		t.Error("set order changed the fingerprint")
	}
	if fingerprint(a) == fingerprint(wf([]any{"A"}, []any{"email", "sms"})) {
		t.Error("a removed trigger event should change the fingerprint")
	}
	if got := a["trigger_events"].([]any)[0]; got != "A" {
		t.Error("the workflow was modified")
	}
}

// A file without is_enabled keeps the workspace's value: the push sends null
// (which the API reads as "keep"), and the missing key is not a difference.
func TestPushWithoutIsEnabledKeepsStoredValue(t *testing.T) {
	api := &fakeAPI{drafts: map[string]map[string]any{}}
	api.save("welcome", map[string]any{"slug": "welcome", "name": "Welcome", "is_enabled": false})
	srv := httptest.NewServer(api.handler(t))
	defer srv.Close()
	client := mgmnt.NewClientWithUrls("token", srv.URL+"/", srv.URL+"/", false)

	local := map[string]any{"slug": "welcome", "name": "Welcome"}
	status, _, err := pushOne(client, "staging", "welcome", local, pushOptions{})
	if err != nil || status != utils.StatusUnchanged || api.pushCount() != 0 {
		t.Fatalf("file without is_enabled: got %q, %v, %d pushes; want unchanged and nothing sent", status, err, api.pushCount())
	}

	local["name"] = "Welcome!"
	status, _, err = pushOne(client, "staging", "welcome", local, pushOptions{})
	if err != nil || status != utils.StatusUpdated {
		t.Fatalf("edited file: got %q, %v; want updated", status, err)
	}
	if v, ok := api.lastBody["is_enabled"]; !ok || v != nil {
		t.Errorf("is_enabled sent as %v (present %v), want an explicit null", v, ok)
	}
	if api.drafts["welcome"]["is_enabled"] != false {
		t.Error("the workflow should still be disabled")
	}
	if _, ok := local["is_enabled"]; ok {
		t.Error("the local workflow was modified")
	}

	local["is_enabled"] = true
	status, _, err = pushOne(client, "staging", "welcome", local, pushOptions{})
	if err != nil || status != utils.StatusUpdated || api.drafts["welcome"]["is_enabled"] != true {
		t.Fatalf("is_enabled true: got %q, %v, stored %v; want updated and enabled", status, err, api.drafts["welcome"]["is_enabled"])
	}
}
