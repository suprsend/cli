package tools

import (
	"context"

	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/cli/pkg/mcpsdk"
)

// markSessionDead is the tool-local entry point for signalling that the
// current MCP session should be torn down (typically after a downstream API
// returns HTTP 401). It dispatches through utils.MarkSessionDead — a function
// variable wired by pkg/mcpserver.init() — so internal/tools never has to
// import pkg/mcpserver directly. That direct import would create an import
// cycle once pkg/mcpserver started depending on internal/tools (for
// BuildTenantTools). In CLI-only builds that don't link pkg/mcpserver, the
// hook is nil and this call is a no-op.
func markSessionDead(ctx context.Context) {
	if utils.MarkSessionDead != nil {
		utils.MarkSessionDead(ctx)
	}
}

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
