package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	log "github.com/sirupsen/logrus"
	"github.com/suprsend/cli/internal/commands/schema"
	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/suprsend-go"
)

func triggerWorkflow(_ context.Context, request mcp.CallToolRequest, workspace, slug string) (*mcp.CallToolResult, error) {
	wfRequestRaw := request.GetArguments()
	tenantId := request.GetString("tenant_id", "")

	actorDistinctId := request.GetString("actor_distinct_id", "")
	recipientDistinctId := request.GetString("recipient_distinct_id", "")

	suprsendClient, err := utils.GetSuprSendWorkspaceClient(workspace)
	if err != nil {
		log.Error("Error getting workspace client: ", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	// add workflow slug to the request body
	wfRequestBody := map[string]any{}
	wfRequestBody["workflow"] = slug

	// if tenant id is present, add it to the request body
	if tenantId != "" {
		wfRequestBody["tenant_id"] = tenantId
	}
	// if actor distinct id is present, add it to the request body as actor.distinct_id
	if actorDistinctId != "" {
		wfRequestBody["actor"] = map[string]any{"distinct_id": actorDistinctId}
	}
	// if recipient distinct id is present, add it to the request body as recipients
	if recipientDistinctId != "" {
		wfRequestBody["recipients"] = []string{recipientDistinctId}
	}
	// Add data to the request body
	wfRequestBody["data"] = wfRequestRaw["data"]
	idempotencyKey := utils.GenerateUUID()
	wf := &suprsend.WorkflowTriggerRequest{
		Body:           wfRequestBody,
		IdempotencyKey: idempotencyKey,
		TenantId:       tenantId,
	}

	resp, err := suprsendClient.Workflows.Trigger(wf)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	responseStruct := map[string]any{
		"idempotency_key": idempotencyKey,
		"status":          resp.StatusCode,
		"success":         resp.Success,
	}
	jsonData, err := json.MarshalIndent(responseStruct, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultStructured(responseStruct, string(jsonData)), nil
}

func listWorkflowsHandler(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	workspace, err := request.RequireString("workspace")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	limit := request.GetInt("limit", 50)
	offset := request.GetInt("offset", 0)
	mode := request.GetString("mode", "live")
	mgmntClient := utils.GetSuprSendMgmntClient()
	workflows, err := mgmntClient.ListWorkflows(workspace, limit, offset, mode)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	jsonData, err := json.MarshalIndent(workflows, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultStructured(workflows, string(jsonData)), nil
}

func RegisterDynamicWorkflowTools(workspace, workflowsFlag string) error {
	workflows := utils.FetchWorkflowsMcp(workspace, workflowsFlag)
	if len(workflows) == 0 {
		return fmt.Errorf("no workflows present in %s workspace", workspace)
	}
	patchSchema := json.RawMessage(`
		{
			"$schema": "https://json-schema.org/draft/2020-12/schema",
			"title": "TenantActorRecipientsSchema",
			"type": "object",
			"properties": {
				"tenant_id": {
				"type": "string",
				"default": "default",
				"description": "Unique identifier for the tenant. Defaults to 'default' if not provided."
				},
				"actor_distinct_id": {
				"type": "string",
				"default": "",
				"description": "Unique identifier for the actor. Defaults to '' if not provided."
				},
				"recipient_distinct_id": {
				"type": "string",
				"description": "Unique identifier for the recipient."
				}
			},
			"required": [
				"recipient_distinct_id"
			]
		}
	`)

	for _, workflow := range workflows {
		slug := workflow.Slug
		if slug == "" {
			continue
		}
		name := workflow.Name
		description := workflow.Description
		mgmntClient := utils.GetSuprSendMgmntClient()
		// if schema is empty, skip
		if workflow.PayloadSchema.Schema == "" {
			continue
		}
		log.Debugf("Getting schema for workflow %s, schema: %s, version: %s", slug, workflow.PayloadSchema.Schema, workflow.PayloadSchema.Version)
		payloadSchema, err := mgmntClient.GetSchema(workspace, workflow.PayloadSchema.Schema, workflow.PayloadSchema.Version)
		if err != nil {
			return fmt.Errorf("failed to get schema: %s", err)
		}
		inputSchema, err := json.Marshal(payloadSchema.JSONSchema)
		if err != nil {
			return fmt.Errorf("failed to marshal schema: %s", err)
		}
		mergedSchema, err := schema.MergeUnderDataAndValidate(string(patchSchema), string(inputSchema))
		if err != nil {
			log.Errorf("failed to create workflow schema for wf %s: %s", name, err)
			continue
		}
		if name == "" {
			continue
		} else {
			name = strings.ReplaceAll(name, " ", "_")
			name = strings.ToLower(name)
		}
		if description == "" {
			description = fmt.Sprintf("Use this tool to trigger workflow with name: \"%s\"", name)
		} else {
			description = fmt.Sprintf("Use this tool to trigger workflow with name: \"%s\" with description:\"%s\"", name, description)
		}
		cleanSlug := strings.ReplaceAll(slug, "-", "_")
		slugLocal := slug
		wfTool := &Tool{
			Name:        "trigger_" + cleanSlug + "_workflow",
			Description: fmt.Sprintf("Trigger workflow: %s - %s", name, description),
			MCPTool: mcp.NewToolWithRawSchema("trigger_"+cleanSlug+"_workflow",
				description,
				mergedSchema,
			),
			Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				return triggerWorkflow(ctx, request, workspace, slugLocal)
			},
		}
		RegisterWorkflow(wfTool)
	}
	return nil
}

func newWorkflowTools() []*Tool {
	list_workflows := &Tool{
		Name:        "workflows.list",
		Description: "Enables listing workflows from a workspace",
		MCPTool: mcp.NewTool("list_workflows",
			mcp.WithDescription("Use this tool to list workflows from a workspace."),
			mcp.WithString("workspace",
				mcp.Description("SuprSend workspace to list workflows from."),
				mcp.Required(),
			),
			mcp.WithNumber("limit",
				mcp.Description("Limit the number of workflows to list."),
				mcp.Required(),
				mcp.DefaultNumber(50),
			),
			mcp.WithNumber("offset",
				mcp.Description("Offset the number of workflows to list."),
				mcp.DefaultNumber(0),
			),
			mcp.WithString("mode",
				mcp.Description("Mode of workflows to list (draft, live), default: live."),
				mcp.Required(),
				mcp.DefaultString("live"),
			),
			mcp.WithString("workspace",
				mcp.Description("SuprSend workspace to list workflows from."),
				mcp.Required(),
				mcp.DefaultString("staging"),
			),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
		),
		Handler: listWorkflowsHandler,
	}

	return []*Tool{list_workflows}
}

func init() {
	for _, t := range newWorkflowTools() {
		RegisterTool(t, "workflows")
	}
}
