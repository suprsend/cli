package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/suprsend/cli/pkg/mcpsdk"
)

// argsFromJSON is a small helper that parses a JSON object into mcpsdk.Args.
// Keeping it local to the test file (per "Test convention" preamble — only
// extract to testutil if duplication crosses ~50 lines).
func argsFromJSON(t *testing.T, raw string) mcpsdk.Args {
	t.Helper()
	a, err := mcpsdk.NewArgs(json.RawMessage(raw))
	if err != nil {
		t.Fatalf("mcpsdk.NewArgs(%q) error: %v", raw, err)
	}
	return a
}

// TestGetTenant_MissingTenantID_ReturnsError is the tracer bullet: confirms
// that the required-argument check fires before any downstream client lookup.
// This path needs no mocked SuprSend client.
func TestGetTenant_MissingTenantID_ReturnsError(t *testing.T) {
	res, err := getTenantHandler(context.Background(), argsFromJSON(t, `{}`))
	if err != nil {
		t.Fatalf("handler returned protocol error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected IsError=true, got %+v", res)
	}
	if !strings.Contains(res.Text, "tenant_id") {
		t.Errorf("expected error text to mention tenant_id, got %q", res.Text)
	}
}

func TestUpsertTenant_MissingTenantID_ReturnsError(t *testing.T) {
	res, err := upsertTenantHandler(context.Background(), argsFromJSON(t, `{}`))
	if err != nil {
		t.Fatalf("handler returned protocol error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected IsError=true, got %+v", res)
	}
	if !strings.Contains(res.Text, "tenant_id") {
		t.Errorf("expected error text to mention tenant_id, got %q", res.Text)
	}
}

// TestUpdatePreference_MissingTenantID_ReturnsError exercises the first
// branch of the preference-update handler — required-arg validation runs
// before any client lookup.
func TestUpdatePreference_MissingTenantID_ReturnsError(t *testing.T) {
	res, err := updateCategoryPreferenceTenant(context.Background(), argsFromJSON(t, `{}`))
	if err != nil {
		t.Fatalf("handler returned protocol error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected IsError=true, got %+v", res)
	}
	if !strings.Contains(res.Text, "tenant_id") {
		t.Errorf("expected error text to mention tenant_id, got %q", res.Text)
	}
}

// TestUpdatePreference_MissingCategory_ReturnsError pins the second
// required-arg branch: tenant_id present but category absent.
func TestUpdatePreference_MissingCategory_ReturnsError(t *testing.T) {
	res, err := updateCategoryPreferenceTenant(context.Background(), argsFromJSON(t, `{"tenant_id": "acme"}`))
	if err != nil {
		t.Fatalf("handler returned protocol error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected IsError=true, got %+v", res)
	}
	if !strings.Contains(res.Text, "category") {
		t.Errorf("expected error text to mention category, got %q", res.Text)
	}
}

// TestUpdatePreference_BadPreferenceType_ReturnsError pins the typed-arg
// branch where preference must be a string. This guards the manual
// type-assertion (handler does not use RequireString for `preference`).
func TestUpdatePreference_BadPreferenceType_ReturnsError(t *testing.T) {
	res, err := updateCategoryPreferenceTenant(
		context.Background(),
		argsFromJSON(t, `{"tenant_id": "acme", "category": "billing", "preference": 42}`),
	)
	if err != nil {
		t.Fatalf("handler returned protocol error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected IsError=true, got %+v", res)
	}
	if !strings.Contains(res.Text, "preference") {
		t.Errorf("expected error text to mention preference, got %q", res.Text)
	}
}

func TestGetDefaultPreference_MissingTenantID_ReturnsError(t *testing.T) {
	res, err := getDefaultPreferenceTenant(context.Background(), argsFromJSON(t, `{}`))
	if err != nil {
		t.Fatalf("handler returned protocol error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected IsError=true, got %+v", res)
	}
	if !strings.Contains(res.Text, "tenant_id") {
		t.Errorf("expected error text to mention tenant_id, got %q", res.Text)
	}
}

// TestNewTenantTools_RegistersExpectedSurface asserts the tool-definition
// shape (names, required args, annotations) without invoking handlers. This
// is the only happy-path coverage we can offer pre-mocking — it guards the
// public protocol surface so accidental rename / required-arg drift is caught.
func TestNewTenantTools_RegistersExpectedSurface(t *testing.T) {
	want := map[string]struct {
		required       []string
		readOnly       bool
		destructive    bool
		idempotent     bool
		hasDescription bool
	}{
		"get_suprsend_tenant":                       {required: []string{"tenant_id"}, readOnly: true, idempotent: true, hasDescription: true},
		"get_suprsend_tenants":                      {required: nil, readOnly: true, idempotent: true, hasDescription: true},
		"upsert_suprsend_tenant":                    {required: []string{"tenant_id"}, destructive: true, hasDescription: true},
		"update_suprsend_tenant_default_preference": {required: []string{"tenant_id", "category", "preference", "visible_to_subscriber", "mandatory_channels", "blocked_channels"}, destructive: true, idempotent: true, hasDescription: true},
		"get_tenant_default_preference":             {required: []string{"tenant_id"}, readOnly: true, idempotent: true, hasDescription: true},
	}

	got := newTenantTools()
	if len(got) != len(want) {
		t.Fatalf("expected %d tools, got %d", len(want), len(got))
	}

	seen := map[string]bool{}
	for _, env := range got {
		if env.Tool == nil {
			t.Fatalf("envelope missing .Tool — ported envelopes must populate it: %+v", env)
		}
		if env.MCPTool.Name != "" || env.Handler != nil {
			t.Errorf("%s: legacy MCPTool/Handler must be zero on ported envelope (got name=%q)", env.Tool.Name, env.MCPTool.Name)
		}
		expect, ok := want[env.Tool.Name]
		if !ok {
			t.Errorf("unexpected tool %q registered", env.Tool.Name)
			continue
		}
		seen[env.Tool.Name] = true
		if expect.hasDescription && env.Tool.Description == "" {
			t.Errorf("%s: missing description", env.Tool.Name)
		}
		if env.Tool.InputSchema == nil || env.Tool.InputSchema.Type != "object" {
			t.Errorf("%s: InputSchema must be a typed object schema", env.Tool.Name)
			continue
		}
		// Required ordering is stable per declaration site — compare as a set.
		gotRequired := map[string]bool{}
		for _, r := range env.Tool.InputSchema.Required {
			gotRequired[r] = true
		}
		for _, r := range expect.required {
			if !gotRequired[r] {
				t.Errorf("%s: required arg %q missing from schema (got %v)", env.Tool.Name, r, env.Tool.InputSchema.Required)
			}
		}
		if env.Tool.Annotations.ReadOnlyHint != expect.readOnly {
			t.Errorf("%s: ReadOnlyHint = %v, want %v", env.Tool.Name, env.Tool.Annotations.ReadOnlyHint, expect.readOnly)
		}
		if env.Tool.Annotations.DestructiveHint != expect.destructive {
			t.Errorf("%s: DestructiveHint = %v, want %v", env.Tool.Name, env.Tool.Annotations.DestructiveHint, expect.destructive)
		}
		if env.Tool.Annotations.IdempotentHint != expect.idempotent {
			t.Errorf("%s: IdempotentHint = %v, want %v", env.Tool.Name, env.Tool.Annotations.IdempotentHint, expect.idempotent)
		}
		// OpenWorldHint is true on every tenant tool — tighten if that ever drifts.
		if !env.Tool.Annotations.OpenWorldHint {
			t.Errorf("%s: OpenWorldHint should be true", env.Tool.Name)
		}
		if env.Tool.Handler == nil {
			t.Errorf("%s: Handler must be set", env.Tool.Name)
		}
	}
	for name := range want {
		if !seen[name] {
			t.Errorf("expected tool %q was not registered", name)
		}
	}
}
