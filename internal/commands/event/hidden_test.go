package event

import (
	"os"
	"path/filepath"
	"testing"
)

// A hidden directory (.git, .idea) or file next to the event directories is
// not read as an event.
func TestReadEventsFromDirSkipsHidden(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "ORDER_PLACED"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ORDER_PLACED", "event.json"), []byte(`{"name":"ORDER_PLACED"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".suprsend-state.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	events, err := ReadEventsFromDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Errorf("got %d events, want 1: %v", len(events), events)
	}
}
