package tools

import (
	"github.com/suprsend/cli/pkg/mcpsdk"
)

// Reactive session close on a revoked token is handled by the
// authExpiryTransport HTTP interceptor (internal/utils), which calls
// utils.MarkSessionDead(req.Context()) on any 401. Since every SuprSend and
// mgmnt client now propagates the request context (suprsend-go v0.10.1 and the
// ctx-threaded mgmnt client), tool handlers no longer need a per-call auth
// check — they just thread ctx, which they already do.

// Tool is the registry envelope used by all internal/tools/*.go files. It
// carries a Type label (used by the --tools selector grammar) alongside a
// runtime-agnostic *mcpsdk.Tool. The embedded *mcpsdk.Tool means callers can
// read Name / Description / etc. directly off the envelope.
//
// Selector is the stable, historical name used by the `--tools` exact-name
// grammar (e.g. "users.get"). It is decoupled from the embedded Tool.Name,
// which is the MCP protocol name surfaced to clients (e.g. "get_suprsend_user")
// and may change. Keeping Selector separate means renaming a protocol-facing
// tool name does not break users' existing `--tools=users.get` invocations.
type Tool struct {
	Type     string
	Selector string
	*mcpsdk.Tool
}

// legacyToolSelectors maps each static tool's MCP protocol name to the stable
// `--tools` selector it has always been addressable by. The migration to the
// official SDK renamed the protocol names; this table preserves the selector
// grammar (`--tools=users.get`, etc.) advertised in the command help so
// existing scripts keep working. Tools absent from this table fall back to
// matching on their protocol name (see RegisterTool).
var legacyToolSelectors = map[string]string{
	// users
	"get_suprsend_user":                       "users.get",
	"upsert_suprsend_user":                    "users.upsert",
	"get_suprsend_user_preferences":           "users.get_preferences",
	"update_suprsend_users_preferences":       "user.update_preferences",
	"update_suprsend_user_channel_preference": "users.update_channel_preference",
	"get_suprsend_user_list_subscriptions":    "users.get_list_subscriptions",
	"get_suprsend_user_objects_subscriptions": "users.get_objects_subscriptions",
	// objects
	"get_suprsend_object":                        "objects.get",
	"upsert_suprsend_object":                     "objects.upsert",
	"get_suprsend_object_preferences":            "objects.get_preferences",
	"update_suprsend_category_preference_object": "objects.update_preferences",
	"update_suprsend_object_channel_preference":  "objects.update_channel_preference",
	"get_suprsend_object_subscriptions":          "object.get_subscriptions",
	"add_suprsend_object_subscriptions":          "object.upsert_subscriptions",
	// tenants
	"get_suprsend_tenant":                       "tenants.get",
	"get_suprsend_tenants":                      "tenants.get_all",
	"upsert_suprsend_tenant":                    "tenants.upsert",
	"update_suprsend_tenant_default_preference": "tenants.update_preferences",
	"get_tenant_default_preference":             "tenants.get_preferences",
	// workflows
	"list_workflows": "workflows.list",
	// documentation
	"search_suprsend_documentation": "documentation.search",
	"fetch_suprsend_documentation":  "documentation.fetch",
}

var (
	toolRegistry     []*Tool
	eventRegistry    []*Tool
	workflowRegistry []*Tool
)

func RegisterTool(t *Tool, toolType string) {
	t.Type = toolType
	// Pin the historical --tools selector when one exists, else fall back to
	// the protocol name so the exact-name grammar still has something to match.
	if sel, ok := legacyToolSelectors[t.Name]; ok {
		t.Selector = sel
	} else if t.Tool != nil {
		t.Selector = t.Name
	}
	toolRegistry = append(toolRegistry, t)
}

func RegisterEvent(t *Tool) {
	t.Type = "event"
	eventRegistry = append(eventRegistry, t)
}

func RegisterWorkflow(t *Tool) {
	t.Type = "workflow"
	workflowRegistry = append(workflowRegistry, t)
}

func GetAllEvents() []*Tool    { return eventRegistry }
func GetAllWorkflows() []*Tool { return workflowRegistry }
func GetAllTools() []*Tool     { return toolRegistry }
