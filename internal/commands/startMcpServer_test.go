/*
Copyright © 2025 SuprSend
*/
package commands

import (
	"testing"

	toolset "github.com/suprsend/cli/internal/tools"
)

// TestGetSelectedTools_LegacySelector verifies that the historical dotted
// selector names (e.g. "users.get") still resolve to the right tool after the
// MCP migration renamed Tool.Name to protocol names (e.g. "get_suprsend_user").
// Users' existing --tools=users.get scripts must keep working.
func TestGetSelectedTools_LegacySelector(t *testing.T) {
	got, err := getSelectedTools("users.get")
	if err != nil {
		t.Fatalf("getSelectedTools returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("getSelectedTools(\"users.get\") returned %d tools, want 1", len(got))
	}
	if got[0].Name != "get_suprsend_user" {
		t.Fatalf("getSelectedTools(\"users.get\") resolved to %q, want protocol name \"get_suprsend_user\"", got[0].Name)
	}
}

// TestGetSelectedTools_EverySelectorRoundTrips verifies that every registered
// static tool is addressable by its Selector and resolves back to itself. This
// guards against the legacy-selector table drifting out of sync with the tools.
func TestGetSelectedTools_EverySelectorRoundTrips(t *testing.T) {
	for _, tool := range toolset.GetAllTools() {
		if tool.Selector == "" {
			t.Errorf("tool %q has an empty Selector", tool.Name)
			continue
		}
		got, err := getSelectedTools(tool.Selector)
		if err != nil {
			t.Errorf("getSelectedTools(%q): %v", tool.Selector, err)
			continue
		}
		found := false
		for _, g := range got {
			if g.Name == tool.Name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("selector %q did not resolve to tool %q", tool.Selector, tool.Name)
		}
	}
}

// TestGetSelectedTools_WildcardAndAllNone verifies the category wildcard and
// the all/none keywords behave as documented.
func TestGetSelectedTools_WildcardAndAllNone(t *testing.T) {
	none, err := getSelectedTools("none")
	if err != nil || len(none) != 0 {
		t.Fatalf("getSelectedTools(\"none\") = %d tools, %v; want 0, nil", len(none), err)
	}

	all, err := getSelectedTools("all")
	if err != nil {
		t.Fatalf("getSelectedTools(\"all\"): %v", err)
	}
	if len(all) != len(toolset.GetAllTools()) {
		t.Fatalf("getSelectedTools(\"all\") = %d tools, want %d", len(all), len(toolset.GetAllTools()))
	}

	users, err := getSelectedTools("users.*")
	if err != nil {
		t.Fatalf("getSelectedTools(\"users.*\"): %v", err)
	}
	if len(users) == 0 {
		t.Fatalf("getSelectedTools(\"users.*\") returned no tools")
	}
	for _, u := range users {
		if u.Type != "users" {
			t.Errorf("users.* returned a tool of type %q (%q)", u.Type, u.Name)
		}
	}
}
