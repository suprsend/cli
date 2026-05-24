package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/cli/pkg/mcpsdk"
)

// TestListWorkflows_MissingWorkspace_ReturnsError covers the
// listWorkflowsHandler required-arg branch — the workspace check fires
// before any mgmnt client lookup, so this path needs no live token and no
// mocked SuprSend client. Mirrors the spirit of events.go's
// missing-distinct_id test: probe the cheapest pre-network failure path to
// pin down the handler's argument-validation contract.
//
// triggerWorkflow has no such pre-network required-arg check (slug is closure-
// captured at registration time, not pulled from args), so an analogous test
// for triggerWorkflow would have to mock the workspace client. We rely
// instead on the higher-level CLI smoke test (list-tools) to catch any wiring
// regression, and on the RegisterDynamicWorkflowToolsFor test below to
// confirm the registration path is panic-safe with no creds.
func TestListWorkflows_MissingWorkspace_ReturnsError(t *testing.T) {
	args, err := mcpsdk.NewArgs(json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("mcpsdk.NewArgs: %v", err)
	}
	res, err := listWorkflowsHandler(context.Background(), args)
	if err != nil {
		t.Fatalf("handler returned protocol error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected IsError=true, got %+v", res)
	}
	if !strings.Contains(res.Text, "workspace") {
		t.Errorf("expected error text to mention workspace, got %q", res.Text)
	}
}

// TestRegisterDynamicWorkflowToolsFor_NoCredsNoSingleton_ReturnsEmpty
// asserts that when ctx has no tenant credentials AND the singleton
// SDKInstance is nil (multi-tenant startup before InitSDK),
// RegisterDynamicWorkflowToolsFor returns a clean error instead of
// panicking. FetchWorkflowsMcpFor returns nil workflows, and MgmntClientFor
// returns nil — the guard surfaces the latter as an explicit error so
// callers don't silently get an empty tool surface.
func TestRegisterDynamicWorkflowToolsFor_NoCredsNoSingleton_ReturnsEmpty(t *testing.T) {
	prevSDK := utils.SDKInstance
	utils.SDKInstance = nil
	t.Cleanup(func() { utils.SDKInstance = prevSDK })

	tools, err := RegisterDynamicWorkflowToolsFor(context.Background(), "ws", "all")
	if err == nil {
		t.Fatalf("expected error when no mgmnt client is available; got tools=%v", tools)
	}
	if len(tools) != 0 {
		t.Fatalf("expected 0 tools when no mgmnt client; got %d", len(tools))
	}
	if !strings.Contains(err.Error(), "no mgmnt client") {
		t.Errorf("expected error to mention missing mgmnt client; got %q", err.Error())
	}
}
