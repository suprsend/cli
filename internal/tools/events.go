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
	suprsend "github.com/suprsend/suprsend-go"
	"golang.org/x/sync/errgroup"
)

// triggerEvent is the shared handler invoked by every dynamically registered
// `trigger_<event_name>_event` tool. It validates distinct_id, builds the
// event property bag from the remaining args, and calls the workspace
// suprsend client's TrackEvent. Auth failures call MarkSessionDead so hosted
// sessions get torn down instead of looping on a revoked token.
func triggerEvent(ctx context.Context, args mcpsdk.Args, workspace, name string) (mcpsdk.Result, error) {
	distinctID, err := args.RequireString("distinct_id")
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	// Build event payload from every arg except distinct_id (which is a
	// routing concern, not a property).
	eventRequestBody := map[string]any{}
	for k, v := range args.Map() {
		if k != "distinct_id" {
			eventRequestBody[k] = v
		}
	}

	log.Debugf("Event request body: %s, distinct_id: %s", eventRequestBody, distinctID)

	suprsendClient, err := utils.GetSuprSendWorkspaceClient(workspace, ctx)
	if err != nil {
		if utils.IsAuthError(err) {
			markSessionDead(ctx)
		}
		return mcpsdk.Result{Text: fmt.Sprintf("Failed to get suprsend client: %v", err), IsError: true}, nil
	}
	event := &suprsend.Event{
		DistinctId: distinctID,
		EventName:  name,
		Properties: eventRequestBody,
	}
	if _, err := suprsendClient.TrackEvent(event); err != nil {
		if utils.IsAuthError(err) {
			markSessionDead(ctx)
		}
		return mcpsdk.Result{Text: fmt.Sprintf("Failed to trigger event: %v", err), IsError: true}, nil
	}
	return mcpsdk.Result{Text: "Event triggered successfully"}, nil
}

// RegisterDynamicEventsToolsFor builds a per-tenant slice of event-trigger
// tools using the mgmnt client on ctx. Used by the hosted MCP server where
// each session has its own credentials and SDKInstance is nil. Returns an
// empty slice (not an error) when no events are selected — callers decide
// whether absence is fatal.
func RegisterDynamicEventsToolsFor(ctx context.Context, workspace, eventsFlag string) ([]*Tool, error) {
	events := utils.FetchEventsMcpFor(ctx, workspace, eventsFlag)
	mgmntClient := utils.MgmntClientFor(ctx)
	if mgmntClient == nil {
		return nil, fmt.Errorf("tools: RegisterDynamicEventsToolsFor: no mgmnt client on ctx")
	}
	tools := make([]*Tool, len(events))

	patchSchema := `{
	"properties":{
		"distinct_id":{
			"type":"string"
		}
	},
	"required":["distinct_id"]
	}`

	g := new(errgroup.Group)
	g.SetLimit(dynamicRegistrationConcurrency)
	for i, event := range events {
		i, event := i, event
		if event.Name == "" || event.PayloadSchema.Schema == "" {
			continue
		}
		g.Go(func() error {
			payloadSchema, err := mgmntClient.GetSchema(workspace, event.PayloadSchema.Schema, event.PayloadSchema.Version)
			if err != nil {
				log.Errorf("event %s: skipping registration — failed to fetch payload schema: %s", event.Name, err)
				return nil
			}
			inputSchemaBytes, err := json.Marshal(payloadSchema.JSONSchema)
			if err != nil {
				log.Errorf("event %s: skipping registration — failed to marshal schema: %s", event.Name, err)
				return nil
			}
			mergedSchema, err := schema.MergeAndValidate(string(inputSchemaBytes), patchSchema)
			if err != nil {
				log.Errorf("event %s: skipping registration — failed to merge schema: %s", event.Name, err)
				return nil
			}
			var inputSchema jsonschema.Schema
			if err := json.Unmarshal(mergedSchema, &inputSchema); err != nil {
				log.Errorf("event %s: skipping registration — failed to parse merged schema: %s", event.Name, err)
				return nil
			}
			// CRITICAL (Issue-18, Machiavelli review): Server.AddTool in the
			// new SDK panics if InputSchema.Type != "object" (verified
			// mcp/server.go:241-260). Schema merge can yield a non-object
			// type if a customer's payload schema is malformed; we skip
			// (loud-log) rather than crash the whole MCP server boot.
			if inputSchema.Type != "object" {
				log.Errorf("event %s: skipping registration — merged schema is not type=object (got %q)", event.Name, inputSchema.Type)
				return nil
			}
			name := strings.ToLower(strings.ReplaceAll(event.Name, " ", "_"))
			description := event.Description
			if description == "" {
				description = fmt.Sprintf("Use this tool to trigger event with name: %q", name)
			} else {
				description = fmt.Sprintf("Use this tool to trigger event with name: %q with description: %q", name, description)
			}
			eventName := name
			tools[i] = &Tool{
				Tool: &mcpsdk.Tool{
					Name:        "trigger_" + eventName + "_event",
					Description: description,
					InputSchema: &inputSchema,
					Handler: func(ctx context.Context, args mcpsdk.Args) (mcpsdk.Result, error) {
						return triggerEvent(ctx, args, workspace, eventName)
					},
				},
			}
			return nil
		})
	}
	// errgroup.Wait never returns an error here — failures are logged and
	// skipped per-event above; registration is best-effort.
	_ = g.Wait()

	out := make([]*Tool, 0, len(tools))
	for _, t := range tools {
		if t != nil {
			t.Type = "event"
			out = append(out, t)
		}
	}
	return out, nil
}

// RegisterDynamicEventsTools is the CLI-compatibility wrapper used by
// startMcpServer.go. Delegates to RegisterDynamicEventsToolsFor with a
// background context (so it resolves the singleton SDKInstance via
// MgmntClientFor's fallback) and appends to the package-level eventRegistry.
func RegisterDynamicEventsTools(workspace, eventsFlag string) error {
	out, err := RegisterDynamicEventsToolsFor(context.Background(), workspace, eventsFlag)
	if err != nil {
		return err
	}
	for _, t := range out {
		eventRegistry = append(eventRegistry, t)
	}
	return nil
}
