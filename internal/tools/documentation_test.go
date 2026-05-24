package tools

import (
	"context"
	"strings"
	"testing"
)

// TestSearchDocs_MissingQuery_ReturnsError is the tracer bullet: confirms the
// required-argument check fires before any outbound HTTP request. This path
// needs no network or mocking — it exits in the first branch of the handler.
func TestSearchDocs_MissingQuery_ReturnsError(t *testing.T) {
	res, err := searchDocsHandler(context.Background(), argsFromJSON(t, `{}`))
	if err != nil {
		t.Fatalf("handler returned protocol error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected IsError=true, got %+v", res)
	}
	if !strings.Contains(res.Text, "query") {
		t.Errorf("expected error text to mention query, got %q", res.Text)
	}
}

// TestFetchDocs_MissingURI_ReturnsError pins the required-arg branch for the
// fetch handler — exits before any outbound HTTP request.
func TestFetchDocs_MissingURI_ReturnsError(t *testing.T) {
	res, err := fetchDocsHandler(context.Background(), argsFromJSON(t, `{}`))
	if err != nil {
		t.Fatalf("handler returned protocol error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected IsError=true, got %+v", res)
	}
	if !strings.Contains(res.Text, "uri") {
		t.Errorf("expected error text to mention uri, got %q", res.Text)
	}
}

// TestNewDocumentationTools_RegistersExpectedSurface asserts the tool-definition
// shape (names, required args, annotations) without invoking handlers. Mirrors
// the tenant_test guard so accidental rename / required-arg drift is caught.
func TestNewDocumentationTools_RegistersExpectedSurface(t *testing.T) {
	want := map[string]struct {
		required       []string
		readOnly       bool
		destructive    bool
		idempotent     bool
		openWorld      bool
		hasDescription bool
	}{
		"search_suprsend_documentation": {required: []string{"query"}, readOnly: true, idempotent: true, openWorld: true, hasDescription: true},
		"fetch_suprsend_documentation":  {required: []string{"uri"}, readOnly: true, idempotent: true, openWorld: true, hasDescription: true},
	}

	got := newDocumentationTools()
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
