package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/cli/pkg/mcpsdk"
)

// TestTriggerEvent_MissingDistinctID_ReturnsError exercises the first
// branch of the triggerEvent handler — the required distinct_id check fires
// before any workspace client lookup, so this path needs no mocked SuprSend
// client (and no live token).
func TestTriggerEvent_MissingDistinctID_ReturnsError(t *testing.T) {
	args, err := mcpsdk.NewArgs(json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("mcpsdk.NewArgs: %v", err)
	}
	res, err := triggerEvent(context.Background(), args, "staging", "user_signed_up")
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

// TestRegisterDynamicEventsToolsFor_NoCredsNoSingleton_ReturnsEmpty asserts
// that when ctx has no tenant credentials AND the singleton SDKInstance is
// nil (multi-tenant startup before InitSDK), RegisterDynamicEventsToolsFor
// returns a clean error instead of panicking. FetchEventsMcpFor returns nil
// events, and MgmntClientFor returns nil — the guard surfaces the latter as
// an explicit error so callers don't silently get an empty tool surface.
func TestRegisterDynamicEventsToolsFor_NoCredsNoSingleton_ReturnsEmpty(t *testing.T) {
	prevSDK := utils.SDKInstance
	utils.SDKInstance = nil
	t.Cleanup(func() { utils.SDKInstance = prevSDK })

	tools, err := RegisterDynamicEventsToolsFor(context.Background(), "ws", "all")
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
