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
type Tool struct {
	Type string
	*mcpsdk.Tool
}

var (
	toolRegistry     []*Tool
	eventRegistry    []*Tool
	workflowRegistry []*Tool
)

func RegisterTool(t *Tool, toolType string) {
	t.Type = toolType
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
