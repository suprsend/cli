package tools

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/suprsend/cli/pkg/mcpsdk"
)

// Tool is the registry envelope used by all internal/tools/*.go files.
//
// During the migration (Phases 4–5), Tool carries BOTH the legacy mark3labs
// fields (MCPTool + Handler, populated by un-ported tool files) AND the new
// runtime-agnostic *mcpsdk.Tool (populated by ported tool files). The CLI's
// startMcpServer.go branches at registration time: if Tool != nil, register
// via mcpsdk/legacy; else use the old mcpServer.AddTool(MCPTool, Handler)
// path. This keeps every commit green.
//
// At the end of Phase 5 (after the last tool file is ported), Task 5.x
// removes MCPTool + Handler and this envelope becomes:
//
//	type Tool struct { Type string; *mcpsdk.Tool }
type Tool struct {
	Type string
	Name string

	// Legacy mark3labs fields — populated by un-ported tool files. Removed
	// in Task 5.x after the last port. Both fields nil when Tool is populated.
	MCPTool mcp.Tool
	Handler server.ToolHandlerFunc

	// New runtime-agnostic field — populated by ported tool files. nil when
	// MCPTool + Handler are populated. After Task 5.x both become exclusive.
	Tool *mcpsdk.Tool
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
