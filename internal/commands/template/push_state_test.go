package template

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/state"
	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/cli/mgmnt"
)

// fakeTemplateAPI serves one template's draft and records variant pushes.
type fakeTemplateAPI struct {
	mu         sync.Mutex
	fields     map[string]any
	variants   map[string]map[string]any // "<channel>/<id>"
	pushed     []string                  // "<channel>/<id>" of each variant push
	saves      int
	live       map[string]map[string]any // committed variants
	commits    int
	fieldSaves int
}

func (f *fakeTemplateAPI) handler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		// v2/<ws>/template/<slug>/...
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		rest := parts[4:]
		switch {
		case len(rest) == 0 && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(f.fields)
		case len(rest) == 0 && r.Method == http.MethodPost:
			f.fieldSaves++
			_ = json.NewDecoder(r.Body).Decode(&f.fields)
		case len(rest) == 1 && rest[0] == "commit" && r.Method == http.MethodPatch:
			f.commits++
			f.live = map[string]map[string]any{}
			for k, v := range f.variants {
				f.live[k] = v
			}
		case len(rest) == 1 && rest[0] == "variant": // list
			from := f.variants
			if r.URL.Query().Get("mode") == "live" {
				from = f.live
			}
			results := []map[string]any{}
			for _, key := range sortedKeys(from) {
				results = append(results, from[key])
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"results": results, "meta": map[string]int{"count": len(results)}})
		case len(rest) == 4 && rest[0] == "channel" && r.Method == http.MethodPost:
			var v map[string]any
			_ = json.NewDecoder(r.Body).Decode(&v)
			key := rest[1] + "/" + rest[3]
			f.pushed = append(f.pushed, key)
			f.store(key, v)
		case rest[0] == "mock_data" || (len(rest) == 2 && rest[1] == "order"):
			w.WriteHeader(http.StatusNotFound)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
		}
	}
}

// store saves a variant with fresh server-managed fields.
func (f *fakeTemplateAPI) store(key string, v map[string]any) {
	f.saves++
	stored := map[string]any{}
	for k, val := range v {
		stored[k] = val
	}
	stored["hash"] = fmt.Sprintf("h%d", f.saves)
	stored["approval_status"] = "auto_approved"
	f.variants[key] = stored
}

// dashboardEdit changes a variant's content as someone in the dashboard would.
func (f *fakeTemplateAPI) dashboardEdit(key, text string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	v := map[string]any{}
	for k, val := range f.variants[key] {
		v[k] = val
	}
	v["content"] = map[string]any{"text": text}
	f.store(key, v)
}

func (f *fakeTemplateAPI) takePushed() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	p := f.pushed
	f.pushed = nil
	return p
}

func editVariant(t *testing.T, dir, rel, text string) {
	t.Helper()
	path := filepath.Join(dir, rel, "variant.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var v map[string]any
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatal(err)
	}
	v["content"] = map[string]any{"text": text}
	data, _ = json.Marshal(v)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPushTemplateUsesState(t *testing.T) {
	api := &fakeTemplateAPI{
		fields:   map[string]any{"slug": "order-shipped", "name": "Order shipped", "description": "", "tags": []any{}, "enabled_channels": []any{"email", "sms"}},
		variants: map[string]map[string]any{},
	}
	api.store("email/default", map[string]any{"id": "default", "channel": "email", "locale": "en", "content": map[string]any{"text": "Your order shipped"}})
	api.store("sms/default", map[string]any{"id": "default", "channel": "sms", "locale": "en", "content": map[string]any{"text": "Shipped"}})
	srv := httptest.NewServer(api.handler(t))
	defer srv.Close()
	client := mgmnt.NewClientWithUrls("token", srv.URL+"/", srv.URL+"/", false)

	// Pull.
	outDir := t.TempDir()
	st, err := state.Load(outDir)
	if err != nil {
		t.Fatal(err)
	}
	draft, exists, err := fetchDraft(client, "staging", "order-shipped")
	if err != nil || !exists {
		t.Fatalf("fetch: %v, exists %v", err, exists)
	}
	stats := &TemplateWriteStats{}
	if err := WriteOneTemplate(draft, outDir, stats); err != nil || stats.Failed > 0 {
		t.Fatalf("write: %v %v", err, stats.Errors)
	}
	RecordPulled(st, "staging", outDir, draft)
	dir := filepath.Join(outDir, "order-shipped")
	opts := PushOptions{State: st}

	push := func(step, wantStatus string, wantPushed ...string) string {
		t.Helper()
		status, detail, err := PushTemplate(client, "staging", "order-shipped", dir, opts)
		if err != nil || status != wantStatus {
			t.Fatalf("%s: got %q (%s), %v; want %q", step, status, detail, err, wantStatus)
		}
		if got := api.takePushed(); fmt.Sprint(got) != fmt.Sprint(wantPushed) && !(len(got) == 0 && len(wantPushed) == 0) {
			t.Fatalf("%s: pushed variants %v, want %v", step, got, wantPushed)
		}
		return detail
	}

	push("right after pull", utils.StatusUnchanged)

	editVariant(t, dir, "sms/default", "Shipped today")
	push("edit sms", utils.StatusUpdated, "sms/default")
	push("again", utils.StatusUnchanged)

	// X62: someone edits the email variant in the dashboard. A push that
	// changed only the sms variant must not send the stale email copy.
	api.dashboardEdit("email/default", "Your order is on its way")
	detail := push("stale email, nothing else changed", utils.StatusSkipped)
	if !strings.Contains(detail, "email/default") {
		t.Errorf("skip detail %q doesn't name the variant", detail)
	}
	editVariant(t, dir, "sms/default", "Shipped!")
	detail = push("edit sms over a stale email", utils.StatusUpdated, "sms/default")
	if !strings.Contains(detail, "skipped") || !strings.Contains(detail, "email/default") {
		t.Errorf("detail %q doesn't say the email variant was skipped", detail)
	}
	if got := api.variants["email/default"]["content"]; fmt.Sprint(got) != "map[text:Your order is on its way]" {
		t.Fatalf("the dashboard edit was overwritten: %v", got)
	}

	editVariant(t, dir, "email/default", "Local email edit")
	_, _, err = PushTemplate(client, "staging", "order-shipped", dir, opts)
	var ce *clierr.CLIError
	if !errors.As(err, &ce) || ce.Code != clierr.CodeAPIConflict || !strings.Contains(ce.Message, "email/default") {
		t.Fatalf("edit on both sides: got %v, want a conflict naming email/default", err)
	}
	if got := api.takePushed(); len(got) != 0 {
		t.Fatalf("a conflicting push sent %v", got)
	}

	opts.Overwrite = true
	push("--overwrite", utils.StatusUpdated, "email/default")

	// The state survives a save and load.
	if err := st.Save(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(outDir, state.FileName)); err != nil {
		t.Fatalf("state file not written: %v", err)
	}
}

// pullFake pulls the fake's template into a new directory and records it.
func pullFake(t *testing.T, client *mgmnt.SS_MgmntClient, slug string) (string, *state.File) {
	t.Helper()
	outDir := t.TempDir()
	st, err := state.Load(outDir)
	if err != nil {
		t.Fatal(err)
	}
	draft, exists, err := fetchDraft(client, "staging", slug)
	if err != nil || !exists {
		t.Fatalf("fetch: %v, exists %v", err, exists)
	}
	stats := &TemplateWriteStats{}
	if err := WriteOneTemplate(draft, outDir, stats); err != nil || stats.Failed > 0 {
		t.Fatalf("write: %v %v", err, stats.Errors)
	}
	RecordPulled(st, "staging", outDir, draft)
	return filepath.Join(outDir, slug), st
}

// Filling in a field that template.json had as null (or adding one it
// lacked) is a local change, not a conflict: nobody touched the workspace.
func TestPushTemplateFieldAdded(t *testing.T) {
	api := &fakeTemplateAPI{
		fields:   map[string]any{"slug": "t1", "name": "T1", "description": "", "tags": []any{}, "enabled_channels": nil},
		variants: map[string]map[string]any{},
	}
	api.store("sms/default", map[string]any{"id": "default", "channel": "sms", "content": map[string]any{"text": "hi"}})
	srv := httptest.NewServer(api.handler(t))
	defer srv.Close()
	client := mgmnt.NewClientWithUrls("token", srv.URL+"/", srv.URL+"/", false)
	dir, st := pullFake(t, client, "t1")
	opts := PushOptions{State: st}

	if status, _, err := PushTemplate(client, "staging", "t1", dir, opts); err != nil || status != utils.StatusUnchanged {
		t.Fatalf("right after pull: %q, %v", status, err)
	}

	path := filepath.Join(dir, "template.json")
	data, _ := os.ReadFile(path)
	var tj map[string]any
	if err := json.Unmarshal(data, &tj); err != nil {
		t.Fatal(err)
	}
	if v, ok := tj["enabled_channels"]; !ok || v != nil {
		t.Fatalf("expected the pulled file to have enabled_channels: null, got %v", tj)
	}
	tj["enabled_channels"] = []any{"sms"}
	data, _ = json.Marshal(tj)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	status, detail, err := PushTemplate(client, "staging", "t1", dir, opts)
	if err != nil || status != utils.StatusUpdated || !strings.Contains(detail, "template fields") {
		t.Fatalf("after setting enabled_channels: %q (%s), %v; want updated", status, detail, err)
	}
	if fmt.Sprint(api.fields["enabled_channels"]) != "[sms]" {
		t.Fatalf("workspace enabled_channels = %v", api.fields["enabled_channels"])
	}
	if status, _, err := PushTemplate(client, "staging", "t1", dir, opts); err != nil || status != utils.StatusUnchanged {
		t.Fatalf("push again: %q, %v", status, err)
	}
}

// A commit publishes the whole draft. A variant that is in the workspace's
// draft but not in the local files, and not live as it stands, must not go
// live unnoticed.
func TestPushTemplateCommitUnseenVariant(t *testing.T) {
	api := &fakeTemplateAPI{
		fields:   map[string]any{"slug": "t1", "name": "T1", "description": "", "tags": []any{}, "enabled_channels": []any{"sms"}},
		variants: map[string]map[string]any{},
	}
	api.store("sms/default", map[string]any{"id": "default", "channel": "sms", "content": map[string]any{"text": "hi"}})
	api.live = map[string]map[string]any{"sms/default": api.variants["sms/default"]}
	srv := httptest.NewServer(api.handler(t))
	defer srv.Close()
	client := mgmnt.NewClientWithUrls("token", srv.URL+"/", srv.URL+"/", false)
	dir, st := pullFake(t, client, "t1")

	// Someone adds a variant in the dashboard.
	api.mu.Lock()
	api.store("sms/promo", map[string]any{"id": "promo", "channel": "sms", "content": map[string]any{"text": "50% off"}})
	api.mu.Unlock()

	// Without --commit the push is unaffected.
	if status, _, err := PushTemplate(client, "staging", "t1", dir, PushOptions{State: st}); err != nil || status != utils.StatusUnchanged {
		t.Fatalf("push without commit: %q, %v", status, err)
	}

	_, _, err := PushTemplate(client, "staging", "t1", dir, PushOptions{State: st, Commit: true})
	var ce *clierr.CLIError
	if !errors.As(err, &ce) || ce.Code != clierr.CodeAPIConflict || !strings.Contains(ce.Message, "sms/promo") {
		t.Fatalf("commit with an unseen draft variant: got %v, want a conflict naming sms/promo", err)
	}
	if api.commits != 0 {
		t.Fatal("the draft was committed")
	}

	status, detail, err := PushTemplate(client, "staging", "t1", dir, PushOptions{State: st, Commit: true, Overwrite: true})
	if err != nil || status != utils.StatusCommitted || !strings.Contains(detail, "sms/promo") {
		t.Fatalf("--overwrite: %q (%s), %v; want committed, naming sms/promo", status, detail, err)
	}

	// Once it is live as it stands, it no longer blocks a commit.
	editVariant(t, dir, "sms/default", "hello")
	status, _, err = PushTemplate(client, "staging", "t1", dir, PushOptions{State: st, Commit: true})
	if err != nil || status != utils.StatusCommitted {
		t.Fatalf("commit after the variant went live: %q, %v", status, err)
	}
}

// #34: pull writes the live version, push compares with the draft. A draft
// variant that differed at pull time is pending work, not a change since.
func TestPushTemplateAfterLivePull(t *testing.T) {
	api := &fakeTemplateAPI{
		fields:   map[string]any{"slug": "t1", "name": "T1", "description": "", "tags": []any{}, "enabled_channels": []any{"sms", "email"}},
		variants: map[string]map[string]any{},
	}
	api.store("sms/default", map[string]any{"id": "default", "channel": "sms", "content": map[string]any{"text": "hi"}})
	api.store("email/default", map[string]any{"id": "default", "channel": "email", "content": map[string]any{"text": "hello"}})
	api.live = map[string]map[string]any{"sms/default": api.variants["sms/default"], "email/default": api.variants["email/default"]}
	api.dashboardEdit("sms/default", "hi, pending") // a draft edit made before the pull
	srv := httptest.NewServer(api.handler(t))
	defer srv.Close()
	client := mgmnt.NewClientWithUrls("token", srv.URL+"/", srv.URL+"/", false)

	// Pull live, as `template pull` does by default.
	outDir := t.TempDir()
	st, err := state.Load(outDir)
	if err != nil {
		t.Fatal(err)
	}
	results, err := FetchTemplates(client, "staging", "live", "t1", FetchOptions{WithDraft: true})
	if err != nil || len(results) != 1 || results[0].Draft == nil {
		t.Fatalf("fetch: %v, %d result(s)", err, len(results))
	}
	stats := &TemplateWriteStats{}
	if err := WriteOneTemplate(results[0], outDir, stats); err != nil || stats.Failed > 0 {
		t.Fatalf("write: %v %v", err, stats.Errors)
	}
	RecordPulled(st, "staging", outDir, results[0])
	dir := filepath.Join(outDir, "t1")

	for _, commit := range []bool{false, true} {
		status, detail, err := PushTemplate(client, "staging", "t1", dir, PushOptions{State: st, Commit: commit})
		if err != nil || status != utils.StatusUnchanged || !strings.Contains(detail, "sms/default") {
			t.Fatalf("commit=%v, untouched files: got %q (%s), %v; want unchanged, naming sms/default", commit, status, detail, err)
		}
	}
	if len(api.takePushed()) != 0 || api.commits != 0 {
		t.Fatal("the pending draft was replaced or committed")
	}

	// Editing another variant doesn't touch the pending one, but a commit
	// would publish it.
	editVariant(t, dir, "email/default", "hello again")
	_, _, err = PushTemplate(client, "staging", "t1", dir, PushOptions{State: st, Commit: true})
	if !clierr.HasCode(err, clierr.CodeAPIConflict) || api.commits != 0 {
		t.Fatalf("commit next to a pending draft variant: got %v, want a conflict", err)
	}
	status, _, err := PushTemplate(client, "staging", "t1", dir, PushOptions{State: st})
	if err != nil || status != utils.StatusUpdated || fmt.Sprint(api.takePushed()) != "[email/default]" {
		t.Fatalf("draft push of the other variant: %q, %v", status, err)
	}

	// Editing the pending variant itself is a conflict.
	editVariant(t, dir, "sms/default", "hi v2")
	_, _, err = PushTemplate(client, "staging", "t1", dir, PushOptions{State: st})
	if !clierr.HasCode(err, clierr.CodeAPIConflict) {
		t.Fatalf("edit over a pending draft variant: got %v, want a conflict", err)
	}
}
