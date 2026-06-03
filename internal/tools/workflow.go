package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	log "github.com/sirupsen/logrus"
	"github.com/suprsend/cli/internal/commands/schema"
	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/cli/pkg/mcpsdk"
	"github.com/suprsend/suprsend-go"
	"golang.org/x/sync/errgroup"
)

// dynamicRegistrationConcurrency caps in-flight schema fetches when the MCP
// server boots with --workflows / --events selecting many resources. Sized
// to amortize HTTP latency without overwhelming the management API.
const dynamicRegistrationConcurrency = 10

// triggerWorkflow is the shared handler invoked by every dynamically
// registered `trigger_<slug>_workflow` tool. It builds the workflow trigger
// payload from args (peeling off tenant_id / actor / recipient as
// first-class routing fields and slotting everything under "data") and calls
// the workspace suprsend client's Workflows.Trigger. Auth failures call
// MarkSessionDead so MCP sessions get torn down instead of looping on a
// revoked token.
func triggerWorkflow(ctx context.Context, args mcpsdk.Args, workspace, slug string) (mcpsdk.Result, error) {
	tenantId := args.GetString("tenant_id", "")
	actorDistinctId := args.GetString("actor_distinct_id", "")
	recipientDistinctId := args.GetString("recipient_distinct_id", "")

	suprsendClient, err := utils.GetSuprSendWorkspaceClient(workspace, ctx)
	if err != nil {
		log.Error("Error getting workspace client: ", err)
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
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
	// Add data to the request body — pulled from args by name so the trigger
	// payload mirrors the workflow input schema field-for-field.
	if data, ok := args.Map()["data"]; ok {
		wfRequestBody["data"] = data
	}
	idempotencyKey := utils.GenerateUUID()
	wf := &suprsend.WorkflowTriggerRequest{
		Body:           wfRequestBody,
		IdempotencyKey: idempotencyKey,
		TenantId:       tenantId,
	}

	resp, err := suprsendClient.Workflows.TriggerWithContext(ctx, wf)
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	responseStruct := map[string]any{
		"idempotency_key": idempotencyKey,
		"status":          resp.StatusCode,
		"success":         resp.Success,
	}
	jsonData, err := json.MarshalIndent(responseStruct, "", "  ")
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}
	return mcpsdk.Result{Text: string(jsonData), Structured: responseStruct}, nil
}

func listWorkflowsHandler(ctx context.Context, args mcpsdk.Args) (mcpsdk.Result, error) {
	workspace, err := args.RequireString("workspace")
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}
	limit := args.GetInt("limit", 50)
	offset := args.GetInt("offset", 0)
	mode := args.GetString("mode", "live")

	mgmntClient := utils.MgmntClientFor(ctx)
	if mgmntClient == nil {
		return mcpsdk.Result{Text: "no mgmnt client available", IsError: true}, nil
	}
	workflows, err := mgmntClient.ListWorkflows(ctx, workspace, limit, offset, mode)
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}
	jsonData, err := json.MarshalIndent(workflows, "", "  ")
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}
	return mcpsdk.Result{Text: string(jsonData), Structured: workflows}, nil
}

// RegisterDynamicWorkflowToolsFor builds a per-tenant slice of
// workflow-trigger tools using the mgmnt client on ctx. Used by
// multi-tenant MCP servers where each session has its own credentials and
// SDKInstance is nil. Returns an empty slice (not an error) when no
// workflows are selected — callers decide whether absence is fatal.
func RegisterDynamicWorkflowToolsFor(ctx context.Context, workspace, workflowsFlag string) ([]*Tool, error) {
	workflows := utils.FetchWorkflowsMcpFor(ctx, workspace, workflowsFlag)
	return registerDynamicWorkflowTools(ctx, workspace, workflows)
}

// registerDynamicWorkflowtools builds the workflow-trigger tools from an
// already-fetched workflow list. Split out from RegisterDynamicWorkflowToolsFor
// so callers that need the workflow list for their own checks (e.g. the CLI's
// "no workflows present" fatal-boot guard) can fetch once and pass the result
// through, avoiding a second GetWorkflows round-trip.
func registerDynamicWorkflowTools(ctx context.Context, workspace string, workflows []utils.WorkflowInfo) ([]*Tool, error) {
	mgmntClient := utils.MgmntClientFor(ctx)
	if mgmntClient == nil {
		return nil, fmt.Errorf("tools: RegisterDynamicWorkflowToolsFor: no mgmnt client on ctx")
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

	tools := make([]*Tool, len(workflows))

	g := new(errgroup.Group)
	g.SetLimit(dynamicRegistrationConcurrency)
	for i, workflow := range workflows {
		if workflow.Slug == "" || workflow.PayloadSchema.Schema == "" {
			continue
		}
		g.Go(func() error {
			log.Debugf("Getting schema for workflow %s, schema: %s, version: %s", workflow.Slug, workflow.PayloadSchema.Schema, workflow.PayloadSchema.Version)
			payloadSchema, err := mgmntClient.GetSchema(ctx, workspace, workflow.PayloadSchema.Schema, workflow.PayloadSchema.Version)
			if err != nil {
				log.Errorf("workflow %s: skipping registration — failed to fetch payload schema: %s", workflow.Slug, err)
				return nil
			}
			inputSchemaBytes, err := json.Marshal(payloadSchema.JSONSchema)
			if err != nil {
				log.Errorf("workflow %s: skipping registration — failed to marshal schema: %s", workflow.Slug, err)
				return nil
			}
			mergedSchema, err := schema.MergeUnderDataAndValidate(string(patchSchema), string(inputSchemaBytes))
			if err != nil {
				log.Errorf("workflow %s: skipping registration — failed to merge schema: %s", workflow.Slug, err)
				return nil
			}
			var inputSchema jsonschema.Schema
			if err := json.Unmarshal(mergedSchema, &inputSchema); err != nil {
				log.Errorf("workflow %s: skipping registration — failed to parse merged schema: %s", workflow.Slug, err)
				return nil
			}
			// CRITICAL: Server.AddTool in the new SDK panics if
			// InputSchema.Type != "object". Schema merge can yield a
			// non-object type if a customer's payload schema is malformed;
			// we skip (loud-log) rather than crash the whole MCP server boot.
			if inputSchema.Type != "object" {
				log.Errorf("workflow %s: skipping registration — merged schema is not type=object (got %q)", workflow.Slug, inputSchema.Type)
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
			toolName := "trigger_" + cleanSlug + "_workflow"
			tools[i] = &Tool{
				Tool: &mcpsdk.Tool{
					Name:        toolName,
					Description: description,
					InputSchema: &inputSchema,
					Handler: func(ctx context.Context, args mcpsdk.Args) (mcpsdk.Result, error) {
						return triggerWorkflow(ctx, args, workspace, slugLocal)
					},
				},
			}
			return nil
		})
	}
	// errgroup.Wait never returns an error here — failures are logged and
	// skipped per-workflow above; registration is best-effort.
	_ = g.Wait()

	out := make([]*Tool, 0, len(tools))
	for _, t := range tools {
		if t != nil {
			t.Type = "workflow"
			out = append(out, t)
		}
	}
	return out, nil
}

// RegisterDynamicWorkflowTools is the CLI-compatibility wrapper used by
// startMcpServer.go. Delegates to RegisterDynamicWorkflowToolsFor with a
// background context (so it resolves the singleton SDKInstance via
// MgmntClientFor's fallback) and appends to the package-level
// workflowRegistry.
func RegisterDynamicWorkflowTools(workspace, workflowsFlag string) error {
	ctx := context.Background()
	// Fetch the workflow list ONCE and reuse it for both the legacy
	// "no workflows in workspace" guard (which the CLI surfaces as a fatal
	// boot error) and the actual tool registration. Previously this called
	// FetchWorkflowsMcp here AND RegisterDynamicWorkflowToolsFor re-fetched via
	// FetchWorkflowsMcpFor — two identical GetWorkflows round-trips on boot.
	workflows := utils.FetchWorkflowsMcp(ctx, workspace, workflowsFlag)
	if len(workflows) == 0 {
		return fmt.Errorf("no workflows present in %s workspace", workspace)
	}
	out, err := registerDynamicWorkflowTools(ctx, workspace, workflows)
	if err != nil {
		return err
	}
	for _, t := range out {
		workflowRegistry = append(workflowRegistry, t)
	}
	return nil
}

func newWorkflowTools() []*Tool {
	list_workflows := &Tool{
		Tool: &mcpsdk.Tool{
			Name: "list_workflows",
			Description: `List notification workflows in a workspace, in either draft or live mode.

mode=live returns the currently-active version of each workflow; mode=draft returns the staged-but-not-yet-promoted version. The two can differ — workflows often have a draft change in flight.

Returns: workflow slug, name, status, category, enabled state, and tags.`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"workspace": {
						Type:        "string",
						Description: "SuprSend workspace to list workflows from.",
						Default:     json.RawMessage(`"staging"`),
					},
					"limit": {
						Type:        "number",
						Description: "Limit the number of workflows to list.",
						Default:     json.RawMessage(`50`),
					},
					"offset": {
						Type:        "number",
						Description: "Offset the number of workflows to list.",
						Default:     json.RawMessage(`0`),
					},
					"mode": {
						Type:        "string",
						Description: "Mode of workflows to list (draft, live), default: live.",
						Default:     json.RawMessage(`"live"`),
					},
				},
				Required: []string{"workspace", "limit", "mode"},
			},
			Annotations: mcpsdk.Annotations{
				ReadOnlyHint:   true,
				IdempotentHint: true,
				OpenWorldHint:  mcpsdk.BoolPtr(true),
			},
			Handler: listWorkflowsHandler,
		},
	}

	return []*Tool{list_workflows}
}

func init() {
	for _, t := range newWorkflowTools() {
		RegisterTool(t, "workflows")
	}
}
