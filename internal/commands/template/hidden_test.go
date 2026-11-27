package template

import (
	"os"
	"path/filepath"
	"testing"
)

// Hidden directories and files inside a template directory are not read as
// variants.
func TestReadTemplateVariantsSkipsHidden(t *testing.T) {
	dir := t.TempDir()
	write := func(rel, body string) {
		t.Helper()
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("sms/default/variant.json", `{"id":"default","channel":"sms","content":{"body":"hi"}}`)
	write(".idea/copy/variant.json", `{"id":"copy","channel":"sms"}`)
	write("sms/.backup/variant.json", `{"id":"backup","channel":"sms"}`)
	write(".suprsend-state.json", `{}`)

	variants, err := readTemplateVariants(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(variants) != 1 || variants[0]["id"] != "default" {
		t.Errorf("got %v, want only the sms/default variant", variants)
	}
}
