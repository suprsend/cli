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
	suprsend "github.com/suprsend/suprsend-go"
	"golang.org/x/sync/errgroup"
)

func triggerEvent(_ context.Context, request mcp.CallToolRequest, workspace, name string) (*mcp.CallToolResult, error) {
	allArgs := request.GetArguments()
	distinctID, ok := allArgs["distinct_id"].(string)
	if !ok {
		return mcp.NewToolResultError("distinct_id is required"), nil
	}
	// create JSON of all other arguments except distinct_id
	eventRequestBody := map[string]any{}
	for k, v := range allArgs {
		if k != "distinct_id" {
			eventRequestBody[k] = v
		}
	}

	log.Debugf("Event request body: %s, distinct_id: %s", eventRequestBody, distinctID)

	suprsendClient, err := utils.GetSuprSendWorkspaceClient(workspace)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get suprsend client: %v", err)), nil
	}
	event := &suprsend.Event{
		DistinctId: distinctID,
		EventName:  name,
		Properties: eventRequestBody,
	}
	_, err = suprsendClient.TrackEvent(event)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to trigger event: %v", err)), nil
	}
	return mcp.NewToolResultText("Event triggered successfully"), nil
}

func RegisterDynamicEventsTools(workspace string, eventsFlag string) error {
	events := utils.FetchEventsMcp(workspace, eventsFlag)
	mgmntClient := utils.GetSuprSendMgmntClient()
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
			inputSchema, err := json.Marshal(payloadSchema.JSONSchema)
			if err != nil {
				log.Errorf("event %s: skipping registration — failed to marshal schema: %s", event.Name, err)
				return nil
			}
			mergedSchema, err := schema.MergeAndValidate(string(inputSchema), patchSchema)
			if err != nil {
				log.Errorf("event %s: skipping registration — failed to merge schema: %s", event.Name, err)
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
				Name: "trigger_" + eventName + "_event",
				MCPTool: mcp.NewToolWithRawSchema("trigger_"+eventName+"_event",
					description,
					mergedSchema,
				),
				Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
					return triggerEvent(ctx, request, workspace, eventName)
				},
			}
			return nil
		})
	}
	_ = g.Wait()

	for _, t := range tools {
		if t != nil {
			RegisterEvent(t)
		}
	}
	return nil
}
