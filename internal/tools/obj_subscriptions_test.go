package tools

import (
	"context"
	"strings"
	"testing"
)

// TestGetObjectSubscriptions_MissingObjectID_ReturnsError is the tracer
// bullet — the required-arg check fires before any client lookup, so this
// path needs no mocked SuprSend client.
func TestGetObjectSubscriptions_MissingObjectID_ReturnsError(t *testing.T) {
	res, err := getObjectSubscriptionsHandler(context.Background(), argsFromJSON(t, `{}`))
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

// TestGetObjectSubscriptions_MissingObjectType_ReturnsError pins the second
// required-arg branch: object_id present but object_type absent.
func TestGetObjectSubscriptions_MissingObjectType_ReturnsError(t *testing.T) {
	res, err := getObjectSubscriptionsHandler(context.Background(), argsFromJSON(t, `{"object_id": "obj-1"}`))
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

func TestAddObjectSubscriptions_MissingObjectID_ReturnsError(t *testing.T) {
	res, err := addObjectSubscriptionsHandler(context.Background(), argsFromJSON(t, `{}`))
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

func TestAddObjectSubscriptions_MissingObjectType_ReturnsError(t *testing.T) {
	res, err := addObjectSubscriptionsHandler(context.Background(), argsFromJSON(t, `{"object_id": "obj-1"}`))
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

// TestAddObjectSubscriptions_MissingRecipients_ReturnsError pins the typed-arg
// branch — recipients is required and validated via raw-map lookup, not
// RequireString, so this guards the manual presence check.
func TestAddObjectSubscriptions_MissingRecipients_ReturnsError(t *testing.T) {
	res, err := addObjectSubscriptionsHandler(
		context.Background(),
		argsFromJSON(t, `{"object_id": "obj-1", "object_type": "project"}`),
	)
	if err != nil {
		t.Fatalf("handler returned protocol error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected IsError=true, got %+v", res)
	}
	if !strings.Contains(strings.ToLower(res.Text), "recipients") {
		t.Errorf("expected error text to mention recipients, got %q", res.Text)
	}
}

// TestNewObjSubscriptionsTools_RegistersExpectedSurface asserts the
// tool-definition shape (names, required args, annotations) without invoking
// handlers. Guards the public protocol surface so accidental rename or
// required-arg drift is caught.
func TestNewObjSubscriptionsTools_RegistersExpectedSurface(t *testing.T) {
	want := map[string]struct {
		required       []string
		readOnly       bool
		destructive    bool
		idempotent     bool
		openWorld      bool
		hasDescription bool
	}{
		"get_suprsend_object_subscriptions": {required: []string{"object_id", "object_type"}, readOnly: true, idempotent: true, openWorld: true, hasDescription: true},
		"add_suprsend_object_subscriptions": {required: []string{"object_id", "object_type", "recipients"}, destructive: true, openWorld: true, hasDescription: true},
	}

	got := newObjSubscriptionsTools()
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
