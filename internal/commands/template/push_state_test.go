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
	mu       sync.Mutex
	fields   map[string]any
	variants map[string]map[string]any // "<channel>/<id>"
	pushed   []string                  // "<channel>/<id>" of each variant push
	saves    int
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
			_ = json.NewDecoder(r.Body).Decode(&f.fields)
		case len(rest) == 1 && rest[0] == "variant": // list
			results := []map[string]any{}
			for _, key := range sortedKeys(f.variants) {
				results = append(results, f.variants[key])
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
