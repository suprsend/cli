package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecide(t *testing.T) {
	base := Entry{Remote: "r0", Local: "l0"}
	tests := []struct {
		name          string
		local, remote string
		base          Entry
		hasBase       bool
		overwrite     bool
		want          Action
	}{
		{"new item", "l1", "", Entry{}, false, false, Create},
		{"same as workspace", "l1", "l1", Entry{}, false, false, Unchanged},
		{"no record, differs", "l1", "r1", Entry{}, false, false, Update},
		{"changed locally only", "l1", "r0", base, true, false, Update},
		// X62/X63: someone edited the workspace after the pull; the local copy is stale.
		{"changed in workspace only", "l0", "r1", base, true, false, Skip},
		{"changed on both sides", "l1", "r1", base, true, false, Conflict},
		{"neither changed, stored differently", "l0", "r0", base, true, false, Unchanged},
		{"deleted in workspace, untouched", "l0", "", base, true, false, Skip},
		{"deleted in workspace, edited", "l1", "", base, true, false, Conflict},
		{"overwrite a conflict", "l1", "r1", base, true, true, Update},
		{"overwrite a skip", "l0", "r1", base, true, true, Update},
		{"overwrite a deletion", "l0", "", base, true, true, Create},
		{"overwrite, same", "l1", "l1", base, true, true, Unchanged},
		// #34: pulled from live while the draft differed; nothing changed since.
		{"pending draft, untouched", "l0", "r0", Entry{Remote: "r0", Local: "l0", Pending: true}, true, false, Pending},
		{"pending draft, edited locally", "l1", "r0", Entry{Remote: "r0", Local: "l0", Pending: true}, true, false, Conflict},
		{"pending draft, draft changed again", "l0", "r1", Entry{Remote: "r0", Local: "l0", Pending: true}, true, false, Skip},
		{"pending draft, overwrite", "l0", "r0", Entry{Remote: "r0", Local: "l0", Pending: true}, true, true, Update},
		{"pending deletion, untouched", "l0", "", Entry{Local: "l0", Pending: true}, true, false, Pending},
		{"pending deletion, edited locally", "l1", "", Entry{Local: "l0", Pending: true}, true, false, Conflict},
		{"pending draft, now equal", "l0", "l0", Entry{Remote: "r0", Local: "l0", Pending: true}, true, false, Unchanged},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Decide(tt.local, tt.remote, tt.base, tt.hasBase, tt.overwrite); got != tt.want {
				t.Errorf("Decide = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestFingerprint(t *testing.T) {
	a := map[string]any{"slug": "welcome", "tags": []any{"x"}, "updated_at": "1"}
	b := map[string]any{"tags": []string{"x"}, "slug": "welcome", "updated_at": "2"}
	if Fingerprint(a, "updated_at") != Fingerprint(b, "updated_at") {
		t.Error("equal values with different ignored fields, key order or slice types should match")
	}
	if Fingerprint(a) == Fingerprint(b) {
		t.Error("updated_at differs and isn't ignored, so the fingerprints should differ")
	}
	if _, ok := a["updated_at"]; !ok {
		t.Error("Fingerprint modified its argument")
	}
}

func TestLoadSave(t *testing.T) {
	dir := t.TempDir()
	f, err := Load(dir, "https://api.example.test/")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, FileName)); !os.IsNotExist(err) {
		t.Error("an unchanged empty state should not be written")
	}

	f.Set("staging", "welcome", Entry{Remote: "r", Local: "l"})
	f.Set("staging", "tpl/variants/email/a", Entry{Remote: "r", Local: "l"})
	f.Set("production", "welcome", Entry{Remote: "r2", Local: "l2"})
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}
	g, err := Load(dir, "https://api.example.test/")
	if err != nil {
		t.Fatal(err)
	}
	if e, ok := g.Get("production", "welcome"); !ok || e.Remote != "r2" {
		t.Errorf("production/welcome = %+v, %v", e, ok)
	}
	g.Delete("staging", "tpl/", true)
	if _, ok := g.Get("staging", "tpl/variants/email/a"); ok {
		t.Error("prefix delete left a key")
	}
	if _, ok := g.Get("staging", "welcome"); !ok {
		t.Error("prefix delete removed another key")
	}

	if err := os.WriteFile(filepath.Join(dir, FileName), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir, "https://api.example.test/"); err == nil {
		t.Error("a corrupt state file should be an error")
	}

	var nilFile *File
	nilFile.Set("staging", "x", Entry{})
	if _, ok := nilFile.Get("staging", "x"); ok {
		t.Error("a nil File has no entries")
	}
	if err := nilFile.Save(); err != nil {
		t.Error(err)
	}
}

// Every account has a "staging" workspace, so entries are kept per API host:
// a record from one host must not apply to the same workspace name on another.
func TestStateIsPerHost(t *testing.T) {
	dir := t.TempDir()
	a, err := Load(dir, "https://api-a.example.test/")
	if err != nil {
		t.Fatal(err)
	}
	a.Set("staging", "welcome", Entry{Remote: "r", Local: "l"})
	if err := a.Save(); err != nil {
		t.Fatal(err)
	}
	b, err := Load(dir, "https://api-b.example.test/")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := b.Get("staging", "welcome"); ok {
		t.Error("an entry recorded for another host should not be found")
	}
	b.Set("staging", "welcome", Entry{Remote: "r2", Local: "l2"})
	b.Delete("staging", "other", false)
	if err := b.Save(); err != nil {
		t.Fatal(err)
	}
	// The same host written without a scheme or slash is the same key.
	again, err := Load(dir, "API-A.example.test")
	if err != nil {
		t.Fatal(err)
	}
	if e, ok := again.Get("staging", "welcome"); !ok || e.Remote != "r" {
		t.Errorf("host a entry = %+v, %v; want the one it saved", e, ok)
	}
}

// A version 1 file had no host; its entries move to the host in use.
func TestStateMigratesVersion1(t *testing.T) {
	dir := t.TempDir()
	old := `{"version":1,"workspaces":{"staging":{"welcome":{"remote":"r","local":"l","pending":true}}}}`
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := Load(dir, "https://api-a.example.test/")
	if err != nil {
		t.Fatal(err)
	}
	if e, ok := f.Get("staging", "welcome"); !ok || !e.Pending || e.Remote != "r" {
		t.Errorf("migrated entry = %+v, %v", e, ok)
	}
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, FileName))
	if strings.Contains(string(data), `"workspaces"`) || !strings.Contains(string(data), `"api-a.example.test"`) {
		t.Errorf("saved file should use the host layout: %s", data)
	}
}
