package tools

import (
	"context"
	"strings"
	"testing"
)

// TestGetUser_MissingDistinctID_ReturnsError is the tracer bullet — the
// required-arg check fires before any client lookup, so this path needs no
// mocked SuprSend client.
func TestGetUser_MissingDistinctID_ReturnsError(t *testing.T) {
	res, err := getUserHandler(context.Background(), argsFromJSON(t, `{}`))
	if err != nil {
		t.Fatalf("handler returned protocol error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected IsError=true, got %+v", res)
	}
	if !strings.Contains(res.Text, "distinct_id") {
		t.Errorf("expected error text to mention distinct_id, got %q", res.Text)
	}
}

func TestUpsertUser_MissingDistinctID_ReturnsError(t *testing.T) {
	res, err := upsertUserHandler(context.Background(), argsFromJSON(t, `{}`))
	if err != nil {
		t.Fatalf("handler returned protocol error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected IsError=true, got %+v", res)
	}
	if !strings.Contains(res.Text, "distinct_id") {
		t.Errorf("expected error text to mention distinct_id, got %q", res.Text)
	}
}

// TestUpsertUser_MissingAction_ReturnsError pins the second required-arg
// branch — distinct_id present but action absent. action validation runs
// before workspace-client construction.
func TestUpsertUser_MissingAction_ReturnsError(t *testing.T) {
	res, err := upsertUserHandler(
		context.Background(),
		argsFromJSON(t, `{"distinct_id": "user-1"}`),
	)
	if err != nil {
		t.Fatalf("handler returned protocol error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected IsError=true, got %+v", res)
	}
	if !strings.Contains(res.Text, "action") {
		t.Errorf("expected error text to mention action, got %q", res.Text)
	}
}

func TestGetUserPreferences_MissingDistinctID_ReturnsError(t *testing.T) {
	res, err := getUserPreferencesHandler(context.Background(), argsFromJSON(t, `{}`))
	if err != nil {
		t.Fatalf("handler returned protocol error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected IsError=true, got %+v", res)
	}
	if !strings.Contains(res.Text, "distinct_id") {
		t.Errorf("expected error text to mention distinct_id, got %q", res.Text)
	}
}

// TestUpdateUserPreference_MissingDistinctIDs_ReturnsError pins the
// typed-arg branch — distinct_ids is required and validated via raw-map
// lookup (not RequireString), so this guards the manual presence check.
func TestUpdateUserPreference_MissingDistinctIDs_ReturnsError(t *testing.T) {
	res, err := updateUserPreference(context.Background(), argsFromJSON(t, `{}`))
	if err != nil {
		t.Fatalf("handler returned protocol error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected IsError=true, got %+v", res)
	}
	if !strings.Contains(res.Text, "distinct_ids") {
		t.Errorf("expected error text to mention distinct_ids, got %q", res.Text)
	}
}

// TestUpdateUserChannelPreference_MissingChannelPreferences_ReturnsError
// pins the typed-arg branch — channel_preferences is required and
// validated via raw-map lookup, so this guards the manual presence check.
func TestUpdateUserChannelPreference_MissingChannelPreferences_ReturnsError(t *testing.T) {
	res, err := updateUserChannelPreferenceHandler(context.Background(), argsFromJSON(t, `{}`))
	if err != nil {
		t.Fatalf("handler returned protocol error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected IsError=true, got %+v", res)
	}
	if !strings.Contains(strings.ToLower(res.Text), "channel_preferences") {
		t.Errorf("expected error text to mention channel_preferences, got %q", res.Text)
	}
}

func TestGetUserListSubscriptions_MissingDistinctID_ReturnsError(t *testing.T) {
	res, err := getUserListSubscriptionsHandler(context.Background(), argsFromJSON(t, `{}`))
	if err != nil {
		t.Fatalf("handler returned protocol error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected IsError=true, got %+v", res)
	}
	if !strings.Contains(res.Text, "distinct_id") {
		t.Errorf("expected error text to mention distinct_id, got %q", res.Text)
	}
}

func TestGetUserObjectsSubscriptions_MissingDistinctID_ReturnsError(t *testing.T) {
	res, err := getUserObjectsSubscriptionsHandler(context.Background(), argsFromJSON(t, `{}`))
	if err != nil {
		t.Fatalf("handler returned protocol error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected IsError=true, got %+v", res)
	}
	if !strings.Contains(res.Text, "distinct_id") {
		t.Errorf("expected error text to mention distinct_id, got %q", res.Text)
	}
}

// TestNewUserTools_RegistersExpectedSurface asserts the tool-definition
// shape (names, required args, annotations) without invoking handlers. Guards
// the public protocol surface so accidental rename / required-arg drift is
// caught.
func TestNewUserTools_RegistersExpectedSurface(t *testing.T) {
	want := map[string]struct {
		required       []string
		readOnly       bool
		destructive    bool
		idempotent     bool
		openWorld      bool
		hasDescription bool
	}{
		"get_suprsend_user":                       {required: []string{"distinct_id"}, readOnly: true, idempotent: true, openWorld: true, hasDescription: true},
		"upsert_suprsend_user":                    {required: []string{"distinct_id", "action"}, destructive: true, openWorld: true, hasDescription: true},
		"get_suprsend_user_preferences":           {required: []string{"distinct_id"}, readOnly: true, idempotent: true, openWorld: true, hasDescription: true},
		"update_suprsend_users_preferences":       {required: []string{"distinct_ids", "channel_preferences", "categories"}, destructive: true, idempotent: true, openWorld: true, hasDescription: true},
		"update_suprsend_user_channel_preference": {required: []string{"distinct_id", "channel_preferences"}, destructive: true, idempotent: true, openWorld: true, hasDescription: true},
		"get_suprsend_user_list_subscriptions":    {required: []string{"distinct_id"}, readOnly: true, idempotent: true, openWorld: true, hasDescription: true},
		"get_suprsend_user_objects_subscriptions": {required: []string{"distinct_id"}, readOnly: true, idempotent: true, openWorld: true, hasDescription: true},
	}

	got := newUserTools()
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
		if env.Tool.Annotations.OpenWorldHint != expect.openWorld {
			t.Errorf("%s: OpenWorldHint = %v, want %v", env.Tool.Name, env.Tool.Annotations.OpenWorldHint, expect.openWorld)
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
