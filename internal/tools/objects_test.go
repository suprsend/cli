package tools

import (
	"context"
	"strings"
	"testing"
)

// TestGetObject_MissingObjectID_ReturnsError is the tracer bullet — the
// required-arg check fires before any client lookup, so this path needs no
// mocked SuprSend client.
func TestGetObject_MissingObjectID_ReturnsError(t *testing.T) {
	res, err := getObjectHandler(context.Background(), argsFromJSON(t, `{}`))
	if err != nil {
		t.Fatalf("handler returned protocol error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected IsError=true, got %+v", res)
	}
	if !strings.Contains(res.Text, "object_id") {
		t.Errorf("expected error text to mention object_id, got %q", res.Text)
	}
}

// TestGetObject_MissingObjectType_ReturnsError pins the second required-arg
// branch: object_id present but object_type absent.
func TestGetObject_MissingObjectType_ReturnsError(t *testing.T) {
	res, err := getObjectHandler(context.Background(), argsFromJSON(t, `{"object_id": "obj-1"}`))
	if err != nil {
		t.Fatalf("handler returned protocol error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected IsError=true, got %+v", res)
	}
	if !strings.Contains(res.Text, "object_type") {
		t.Errorf("expected error text to mention object_type, got %q", res.Text)
	}
}

func TestUpsertObject_MissingObjectID_ReturnsError(t *testing.T) {
	res, err := upsertObjectHandler(context.Background(), argsFromJSON(t, `{}`))
	if err != nil {
		t.Fatalf("handler returned protocol error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected IsError=true, got %+v", res)
	}
	if !strings.Contains(res.Text, "object_id") {
		t.Errorf("expected error text to mention object_id, got %q", res.Text)
	}
}

// TestUpsertObject_MissingAction_ReturnsError exercises the third required
// branch (object_id + object_type present, action absent). action validation
// runs before workspace-client construction.
func TestUpsertObject_MissingAction_ReturnsError(t *testing.T) {
	res, err := upsertObjectHandler(
		context.Background(),
		argsFromJSON(t, `{"object_id": "obj-1", "object_type": "project"}`),
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

func TestGetObjectPreferences_MissingObjectID_ReturnsError(t *testing.T) {
	res, err := getObjectPreferences(context.Background(), argsFromJSON(t, `{}`))
	if err != nil {
		t.Fatalf("handler returned protocol error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected IsError=true, got %+v", res)
	}
	if !strings.Contains(res.Text, "object_id") {
		t.Errorf("expected error text to mention object_id, got %q", res.Text)
	}
}

func TestUpdateObjectCategoryPreference_MissingCategory_ReturnsError(t *testing.T) {
	res, err := updateObjectCategoryPreference(
		context.Background(),
		argsFromJSON(t, `{"object_id": "obj-1", "object_type": "project"}`),
	)
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

// TestUpdateObjectChannelPreference_MissingChannelPreferences_ReturnsError
// pins the typed-arg branch — channel_preferences is required and validated
// via raw-map lookup (not RequireString), so this guards the manual presence
// check.
func TestUpdateObjectChannelPreference_MissingChannelPreferences_ReturnsError(t *testing.T) {
	res, err := updateObjectChannelPreferenceHandler(
		context.Background(),
		argsFromJSON(t, `{"object_id": "obj-1", "object_type": "project"}`),
	)
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

// TestNewObjectsTools_RegistersExpectedSurface asserts the tool-definition
// shape (names, required args, annotations) without invoking handlers. Guards
// the public protocol surface so accidental rename / required-arg drift is
// caught.
func TestNewObjectsTools_RegistersExpectedSurface(t *testing.T) {
	want := map[string]struct {
		required       []string
		readOnly       bool
		destructive    bool
		idempotent     bool
		openWorld      bool
		hasDescription bool
	}{
		"get_suprsend_object":                        {required: []string{"object_id", "object_type"}, readOnly: true, idempotent: true, openWorld: true, hasDescription: true},
		"upsert_suprsend_object":                     {required: []string{"object_id", "object_type", "action"}, destructive: true, openWorld: true, hasDescription: true},
		"get_suprsend_object_preferences":            {required: []string{"object_id", "object_type"}, readOnly: true, idempotent: true, openWorld: true, hasDescription: true},
		"update_suprsend_category_preference_object": {required: []string{"object_id", "object_type", "category", "preference"}, destructive: true, idempotent: true, openWorld: true, hasDescription: true},
		"update_suprsend_object_channel_preference":  {required: []string{"object_id", "object_type", "channel_preferences"}, destructive: true, idempotent: true, openWorld: true, hasDescription: true},
	}

	got := newObjectTools()
	if len(got) != len(want) {
		t.Fatalf("expected %d tools, got %d", len(want), len(got))
	}

	seen := map[string]bool{}
	for _, env := range got {
		if env.Tool == nil {
			t.Fatalf("envelope missing .Tool — ported envelopes must populate it: %+v", env)
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
