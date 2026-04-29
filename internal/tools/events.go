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
	for _, event := range events {
		name := event.Name
		description := event.Description
		mgmntClient := utils.GetSuprSendMgmntClient()
		payloadSchema, err := mgmntClient.GetSchema(workspace, event.PayloadSchema.Schema, event.PayloadSchema.Version)
		if err != nil {
			return fmt.Errorf("failed to get schema: %s", err)
		}
		inputSchema, err := json.Marshal(payloadSchema.JSONSchema)
		if err != nil {
			return fmt.Errorf("failed to marshal schema: %s", err)
		}
		patchSchema := `{
		"properties":{
			"distinct_id":{
				"type":"string"
			}
		},
		"required":["distinct_id"]
		}`
		mergedSchema, err := schema.MergeAndValidate(string(inputSchema), patchSchema)
		if err != nil {
			log.Errorf("failed to create event schema for event %s: %s", name, err)
			continue
		}
		if name == "" {
			continue
		} else {
			// clean up the name, replace all spaces and convert to lowercase
			name = strings.ReplaceAll(name, " ", "_")
			name = strings.ToLower(name)
		}
		// if description is empty, don't add it to the description
		if description == "" {
			description = fmt.Sprintf("Use this tool to trigger event with name: \"%s\"", name)
		} else {
			description = fmt.Sprintf("Use this tool to trigger event with name: \"%s\" with description: \"%s\"", name, description)
		}
		eventName := name
		eventTool := &Tool{
			Name: "trigger_" + eventName + "_event",
			MCPTool: mcp.NewToolWithRawSchema("trigger_"+eventName+"_event",
				description,
				mergedSchema,
			),
			Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				return triggerEvent(ctx, request, workspace, eventName)
			},
		}
		RegisterEvent(eventTool)
	}
	return nil
}
