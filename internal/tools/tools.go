package tools

import "github.com/suprsend/cli/pkg/mcpsdk"

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
