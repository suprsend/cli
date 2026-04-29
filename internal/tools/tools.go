package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

type Tool struct {
	Type    string
	Name    string
	MCPTool mcp.Tool
	Handler func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error)
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

func GetAllEvents() []*Tool {
	return eventRegistry
}

func GetAllWorkflows() []*Tool {
	return workflowRegistry
}

func GetAllTools() []*Tool {
	return toolRegistry
}
