package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadDirSkipsHidden(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"en.json", ".suprsend-state.json", ".DS_Store", "._en.json"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"welcome", ".git", ".idea"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if len(names) != 2 || names[0] != "en.json" || names[1] != "welcome" {
		t.Errorf("ReadDir = %v, want [en.json welcome]", names)
	}
}
