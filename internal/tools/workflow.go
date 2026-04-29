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
	"golang.org/x/sync/errgroup"
)

// dynamicRegistrationConcurrency caps in-flight schema fetches when the MCP
// server boots with --workflows / --events selecting many resources. Sized
// to amortize HTTP latency without overwhelming the management API.
const dynamicRegistrationConcurrency = 10

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

	mgmntClient := utils.GetSuprSendMgmntClient()
	tools := make([]*Tool, len(workflows))

	g := new(errgroup.Group)
	g.SetLimit(dynamicRegistrationConcurrency)
	for i, workflow := range workflows {
		i, workflow := i, workflow
		if workflow.Slug == "" || workflow.PayloadSchema.Schema == "" {
			continue
		}
		g.Go(func() error {
			log.Debugf("Getting schema for workflow %s, schema: %s, version: %s", workflow.Slug, workflow.PayloadSchema.Schema, workflow.PayloadSchema.Version)
			payloadSchema, err := mgmntClient.GetSchema(workspace, workflow.PayloadSchema.Schema, workflow.PayloadSchema.Version)
			if err != nil {
				log.Errorf("workflow %s: skipping registration — failed to fetch payload schema: %s", workflow.Slug, err)
				return nil
			}
			inputSchema, err := json.Marshal(payloadSchema.JSONSchema)
			if err != nil {
				log.Errorf("workflow %s: skipping registration — failed to marshal schema: %s", workflow.Slug, err)
				return nil
			}
			mergedSchema, err := schema.MergeUnderDataAndValidate(string(patchSchema), string(inputSchema))
			if err != nil {
				log.Errorf("workflow %s: skipping registration — failed to merge schema: %s", workflow.Slug, err)
				return nil
			}
			name := workflow.Name
			if name == "" {
				return nil
			}
			name = strings.ToLower(strings.ReplaceAll(name, " ", "_"))
			description := workflow.Description
			if description == "" {
				description = fmt.Sprintf("Use this tool to trigger workflow with name: %q", name)
			} else {
				description = fmt.Sprintf("Use this tool to trigger workflow with name: %q with description: %q", name, description)
			}
			cleanSlug := strings.ReplaceAll(workflow.Slug, "-", "_")
			slugLocal := workflow.Slug
			tools[i] = &Tool{
				Name: "trigger_" + cleanSlug + "_workflow",
				MCPTool: mcp.NewToolWithRawSchema("trigger_"+cleanSlug+"_workflow",
					description,
					mergedSchema,
				),
				Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
					return triggerWorkflow(ctx, request, workspace, slugLocal)
				},
			}
			return nil
		})
	}
	// errgroup.Wait never returns an error here — failures are logged and
	// skipped per-workflow above; registration is best-effort.
	_ = g.Wait()

	for _, t := range tools {
		if t != nil {
			RegisterWorkflow(t)
		}
	}
	return nil
}

func newWorkflowTools() []*Tool {
	list_workflows := &Tool{
		Name:        "workflows.list",
		MCPTool: mcp.NewTool("list_workflows",
			mcp.WithDescription(`List notification workflows in a workspace, in either draft or live mode.

mode=live returns the currently-active version of each workflow; mode=draft returns the staged-but-not-yet-promoted version. The two can differ — workflows often have a draft change in flight.

Returns: workflow slug, name, status, category, enabled state, and tags.`),
			mcp.WithString("workspace",
				mcp.Description("SuprSend workspace to list workflows from."),
				mcp.Required(),
				mcp.DefaultString("staging"),
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
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
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
