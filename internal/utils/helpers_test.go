package utils

import (
	"context"
	"testing"
)

// TestFetchEventsMcpFor_NoCredsNoSingleton_ReturnsNil asserts that when ctx has
// no tenant credentials AND the singleton SDKInstance is nil (multi-tenant
// startup), FetchEventsMcpFor returns nil instead of panicking or returning a
// bogus result. This is the safety net that lets a multi-tenant MCP server
// skip dynamic tool registration cleanly when called too early or in the
// wrong mode.
func TestFetchEventsMcpFor_NoCredsNoSingleton_ReturnsNil(t *testing.T) {
	resetSDKGlobals(t)
	SDKInstance = nil

	got := FetchEventsMcpFor(context.Background(), "ws", "all")
	if got != nil {
		t.Fatalf("expected nil with no creds and nil SDKInstance; got %#v", got)
	}
}

// TestFetchWorkflowsMcpFor_NoCredsNoSingleton_ReturnsNil mirrors the events
// case for workflows.
func TestFetchWorkflowsMcpFor_NoCredsNoSingleton_ReturnsNil(t *testing.T) {
	resetSDKGlobals(t)
	SDKInstance = nil

	got := FetchWorkflowsMcpFor(context.Background(), "ws", "all")
	if got != nil {
		t.Fatalf("expected nil with no creds and nil SDKInstance; got %#v", got)
	}
}

// TestFetchEventsMcpFor_NoneSelector_ReturnsNil asserts the selector short-
// circuit still applies on the context-aware variant: passing "none" (or empty)
// must return nil without consulting any client at all.
func TestFetchEventsMcpFor_NoneSelector_ReturnsNil(t *testing.T) {
	resetSDKGlobals(t)
	SDKInstance = nil

	if got := FetchEventsMcpFor(context.Background(), "ws", "none"); got != nil {
		t.Fatalf("expected nil for 'none' selector; got %#v", got)
	}
	if got := FetchEventsMcpFor(context.Background(), "ws", ""); got != nil {
		t.Fatalf("expected nil for empty selector; got %#v", got)
	}
}

// TestFetchWorkflowsMcpFor_NoneSelector_ReturnsNil mirrors the events case for
// workflows.
func TestFetchWorkflowsMcpFor_NoneSelector_ReturnsNil(t *testing.T) {
	resetSDKGlobals(t)
	SDKInstance = nil

	if got := FetchWorkflowsMcpFor(context.Background(), "ws", "none"); got != nil {
		t.Fatalf("expected nil for 'none' selector; got %#v", got)
	}
	if got := FetchWorkflowsMcpFor(context.Background(), "ws", ""); got != nil {
		t.Fatalf("expected nil for empty selector; got %#v", got)
	}
}
