package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"
	suprsend "github.com/suprsend/suprsend-go"
)

type WorkflowPayloadSchema struct {
	Schema string `json:"schema"`
	// version_no is optional and can be nil
	Version string `json:"version_no,omitempty"`
}

type WorkflowInfo struct {
	Slug          string
	Name          string
	Description   string
	PayloadSchema WorkflowPayloadSchema
}

func GenerateUUID() string {
	return uuid.New().String()
}

func FetchWorkflowsMcp(workspace, workflowsFlag string) []WorkflowInfo {
	if workflowsFlag == "none" {
		return nil
	}
	var workflowsToBeFetched []string
	if workflowsFlag != "all" {
		workflowsToBeFetched = strings.Split(workflowsFlag, ",")
		for i, workflow := range workflowsToBeFetched {
			workflowsToBeFetched[i] = strings.TrimSpace(workflow)
		}
	}

	if len(workflowsToBeFetched) == 0 && workflowsFlag != "all" {
		return nil
	}
	mgmntClient := GetSuprSendMgmntClient()
	if mgmntClient == nil {
		return nil
	}
	workflowsResp, err := mgmntClient.GetWorkflows(workspace, "live")
	if err != nil {
		return nil
	}

	var result []WorkflowInfo
	for _, workflow := range workflowsResp.Results {
		workflowMap, ok := workflow.(map[string]any)
		if !ok {
			continue
		}
		workflowInfo := WorkflowInfo{}
		if slug, ok := workflowMap["slug"].(string); ok {
			workflowInfo.Slug = slug
		}
		if name, ok := workflowMap["name"].(string); ok {
			workflowInfo.Name = name
		} else {
			workflowInfo.Name = ""
		}
		if description, ok := workflowMap["description"].(string); ok {
			workflowInfo.Description = description
		} else {
			workflowInfo.Description = ""
		}
		if payloadSchema, ok := workflowMap["payload_schema"].(map[string]any); ok {
			workflowInfo.PayloadSchema.Schema = payloadSchema["schema"].(string)
			if versionNo, ok := payloadSchema["version_no"].(string); ok {
				workflowInfo.PayloadSchema.Version = versionNo
			} else {
				workflowInfo.PayloadSchema.Version = ""
			}
		}

		if workflowsFlag == "all" || slices.Contains(workflowsToBeFetched, workflowInfo.Name) {
			result = append(result, workflowInfo)
		}
	}
	return result
}

type EventPayloadSchema struct {
	Schema string `json:"schema"`
	// version_no is optional and can be nil
	Version string `json:"version_no,omitempty"`
}

type EventInfo struct {
	Name          string
	Description   string
	PayloadSchema EventPayloadSchema
}

func FetchEventsMcp(workspace string, eventsFlag string) []EventInfo {
	// eventFlag can be `none` or `all` or a comma separated list of event slugs
	if eventsFlag == "none" {
		return nil
	}
	var eventsToBeFetched []string
	if eventsFlag != "all" {
		eventsToBeFetched = strings.Split(eventsFlag, ",")
		// trim the events
		for i, event := range eventsToBeFetched {
			eventsToBeFetched[i] = strings.TrimSpace(event)
		}
	}

	if len(eventsToBeFetched) == 0 && eventsFlag != "all" {
		return nil
	}

	mgmntClient := GetSuprSendMgmntClient()
	if mgmntClient == nil {
		return nil
	}
	eventsResp, err := mgmntClient.GetEvents(workspace)
	if err != nil {
		return nil
	}
	var result []EventInfo
	for _, event := range eventsResp.Results {
		eventMap, ok := event.(map[string]any)
		if !ok {
			continue
		}
		eventInfo := EventInfo{}
		if name, ok := eventMap["name"].(string); ok {
			eventInfo.Name = name
		} else {
			eventInfo.Name = ""
		}
		if description, ok := eventMap["description"].(string); ok {
			eventInfo.Description = description
		} else {
			eventInfo.Description = ""
		}
		if payloadSchema, ok := eventMap["payload_schema"].(map[string]any); ok {
			eventInfo.PayloadSchema.Schema = payloadSchema["schema"].(string)
			// check if version_no is nil
			if versionNo, ok := payloadSchema["version_no"].(string); ok {
				eventInfo.PayloadSchema.Version = versionNo
			} else {
				eventInfo.PayloadSchema.Version = ""
			}
		}

		// add th event to the result only if either eventsFlag is "all" or the event is in the eventsToBeFetched list
		if eventsFlag == "all" || slices.Contains(eventsToBeFetched, eventInfo.Name) {
			result = append(result, eventInfo)
		}
	}
	return result
}

func GetExtensionForLanguage(language string) string {
	extMap := map[string]string{
		"python":         ".py",
		"java":           ".java",
		"typescript":     ".ts",
		"typescript-zod": ".ts",
		"go":             ".go",
		"kotlin":         ".kt",
		"swift":          ".swift",
		"dart":           ".dart",
	}
	return extMap[language]
}

func GetStringPtr(m map[string]any, key string) *string {
	if val, ok := m[key].(string); ok {
		return &val
	}
	return nil
}

func GetMap(m map[string]any, key string) map[string]any {
	if val, ok := m[key].(map[string]any); ok {
		return val
	}
	return nil
}

func StringSchema(desc string) map[string]any {
	return map[string]any{
		"type":        "string",
		"description": desc,
	}
}

func BoolSchema(desc string) map[string]any {
	return map[string]any{
		"type":        "boolean",
		"description": desc,
	}
}

func ArraySchema(desc string) map[string]any {
	return map[string]any{
		"type":        "array",
		"description": desc,
		"items": map[string]any{
			"type": "string",
		},
	}
}

func RequiresKey(action string) bool {
	actions := map[string]bool{
		"set": true, "append": true, "increment": true, "unset": true, "remove": true,
	}
	return actions[action]
}

func HandleObjectAction(ctx context.Context, objectInstance suprsend.ObjectEdit, action, key, value string, slack_details map[string]interface{}, identity_provider string, objectIdentifier suprsend.ObjectIdentifier, workspace string) (string, error) {
	var err error
	var out string

	suprsend_client, err := GetSuprSendWorkspaceClient(workspace)
	if err != nil {
		return "", err
	}

	switch action {
	case "upsert":
		if key != "" && value != "" {
			objectInstance.Set(map[string]any{key: value})
			out = fmt.Sprintf("Key set successfully for object with object_id: %s for key: %s and value: %s", objectIdentifier.Id, key, value)
		} else {
			out = "User upserted successfully with object_id: " + objectIdentifier.Id
		}

		obj_edit_request := suprsend.ObjectEditRequest{
			Identifier:   &objectIdentifier,
			EditInstance: objectInstance,
			Payload:      map[string]any{key: value},
		}

		_, err = suprsend_client.Objects.Edit(ctx, obj_edit_request)
	case "remove":
		objectInstance.Remove(map[string]any{key: value})
		out = fmt.Sprintf("Key removed successfully from user with object_id: %s for key: %s and value: %s", objectIdentifier.Id, key, value)
	case "set":
		objectInstance.Set(map[string]any{key: value})
		out = fmt.Sprintf("Key set successfully for user with object_id: %s for key: %s and value: %s", objectIdentifier.Id, key, value)
	case "unset":
		objectInstance.Unset([]string{key})
		out = fmt.Sprintf("Key unset successfully for user with object_id: %s for key: %s and value: %s", objectIdentifier.Id, key, value)
	case "append":
		objectInstance.Append(map[string]any{key: value})
		out = fmt.Sprintf("Key appended successfully for user with object_id: %s for key: %s and value: %s", objectIdentifier.Id, key, value)
	case "increment":
		objectInstance.Increment(map[string]any{key: value})
		out = fmt.Sprintf("Key incremented successfully for user with object_id: %s for key: %s and value: %s", objectIdentifier.Id, key, value)
	case "add_email":
		objectInstance.AddEmail(value)
		out = fmt.Sprintf("Email added successfully for user with object_id: %s for key: %s and value: %s", objectIdentifier.Id, key, value)
	case "set_preferred_language":
		objectInstance.SetPreferredLanguage(value)
		out = fmt.Sprintf("Preferred language set successfully for user with object_id: %s for key: %s and value: %s", objectIdentifier.Id, key, value)
	case "set_timezone":
		objectInstance.SetTimezone(value)
		out = fmt.Sprintf("Timezone set successfully for user with object_id: %s for key: %s and value: %s", objectIdentifier.Id, key, value)
	case "remove_email":
		objectInstance.RemoveEmail(value)
		out = fmt.Sprintf("Email removed successfully for user with object_id: %s for key: %s and value: %s", objectIdentifier.Id, key, value)
	case "add_sms":
		objectInstance.AddSms(value)
		out = fmt.Sprintf("SMS added successfully for user with object_id: %s for key: %s and value: %s", objectIdentifier.Id, key, value)
	case "remove_sms":
		objectInstance.RemoveSms(value)
		out = fmt.Sprintf("SMS removed successfully for user with object_id: %s for key: %s and value: %s", objectIdentifier.Id, key, value)
	case "add_whatsapp":
		objectInstance.AddWhatsapp(value)
		out = fmt.Sprintf("Whatsapp added successfully for user with object_id: %s for key: %s and value: %s", objectIdentifier.Id, key, value)
	case "remove_whatsapp":
		objectInstance.RemoveWhatsapp(value)
		out = fmt.Sprintf("Whatsapp removed successfully for user with object_id: %s for key: %s and value: %s", objectIdentifier.Id, key, value)
	case "add_androidpush":
		objectInstance.AddAndroidpush(value, identity_provider)
		out = fmt.Sprintf("Android push added successfully for user with object_id: %s for key: %s and value: %s and identity_provider: %s", objectIdentifier.Id, key, value, identity_provider)
	case "remove_androidpush":
		objectInstance.RemoveAndroidpush(value, identity_provider)
		out = fmt.Sprintf("Android push removed successfully for user with object_id: %s for key: %s and value: %s and identity_provider: %s", objectIdentifier.Id, key, value, identity_provider)
	case "add_iospush":
		objectInstance.AddIospush(value, identity_provider)
		out = fmt.Sprintf("iOS push added successfully for user with object_id: %s for key: %s and value: %s and identity_provider: %s", objectIdentifier.Id, key, value, identity_provider)
	case "remove_iospush":
		objectInstance.RemoveIospush(value, identity_provider)
		out = fmt.Sprintf("iOS push removed successfully for user with object_id: %s for key: %s and value: %s and identity_provider: %s", objectIdentifier.Id, key, value, identity_provider)
	case "add_slack", "remove_slack":
		payload, slackOut, err := prepareSlackPayload(slack_details)
		if err != nil {
			return "", err
		}
		if action == "add_slack" {
			objectInstance.AddSlack(payload)
		} else {
			objectInstance.RemoveSlack(payload)
		}
		out = fmt.Sprintf(slackOut, objectIdentifier.Id, value)
	}

	return out, err
}

func HandleUserAction(ctx context.Context, userInstance suprsend.UserEdit, action, key, value string, slack_details map[string]interface{}, identity_provider, distinct_id string, workspace string) (string, error) {
	var err error
	var out string

	suprsend_client, err := GetSuprSendWorkspaceClient(workspace)
	if err != nil {
		return "", err
	}

	switch action {
	case "upsert":
		if key != "" && value != "" {
			userInstance.Set(map[string]any{key: value})
			out = fmt.Sprintf("Key set successfully for user with distinct_id: %s for key: %s and value: %s", distinct_id, key, value)
		} else {
			out = "User upserted successfully with distinct_id: " + distinct_id
		}
		_, err = suprsend_client.Users.AsyncEdit(ctx, userInstance)
	case "remove":
		userInstance.Remove(map[string]any{key: value})
		out = fmt.Sprintf("Key removed successfully from user with distinct_id: %s for key: %s and value: %s", distinct_id, key, value)
	case "set":
		userInstance.Set(map[string]any{key: value})
		out = fmt.Sprintf("Key set successfully for user with distinct_id: %s for key: %s and value: %s", distinct_id, key, value)
	case "unset":
		userInstance.Unset([]string{key})
		out = fmt.Sprintf("Key unset successfully for user with distinct_id: %s for key: %s and value: %s", distinct_id, key, value)
	case "append":
		userInstance.Append(map[string]any{key: value})
		out = fmt.Sprintf("Key appended successfully for user with distinct_id: %s for key: %s and value: %s", distinct_id, key, value)
	case "increment":
		if _, err := strconv.Atoi(value); err != nil {
			return "", errors.New("value must be an integer")
		}
		userInstance.Increment(map[string]any{key: value})
		out = fmt.Sprintf("Key incremented successfully for user with distinct_id: %s for key: %s and value: %s", distinct_id, key, value)
	case "add_email":
		userInstance.AddEmail(value)
		out = fmt.Sprintf("Email added successfully for user with distinct_id: %s for key: %s and value: %s", distinct_id, key, value)
	case "set_preferred_language":
		userInstance.SetPreferredLanguage(value)
		out = fmt.Sprintf("Preferred language set successfully for user with distinct_id: %s for key: %s and value: %s", distinct_id, key, value)
	case "set_timezone":
		userInstance.SetTimezone(value)
		out = fmt.Sprintf("Timezone set successfully for user with distinct_id: %s for key: %s and value: %s", distinct_id, key, value)
	case "remove_email":
		userInstance.RemoveEmail(value)
		out = fmt.Sprintf("Email removed successfully for user with distinct_id: %s for key: %s and value: %s", distinct_id, key, value)
	case "add_sms":
		userInstance.AddSms(value)
		out = fmt.Sprintf("SMS added successfully for user with distinct_id: %s for key: %s and value: %s", distinct_id, key, value)
	case "remove_sms":
		userInstance.RemoveSms(value)
		out = fmt.Sprintf("SMS removed successfully for user with distinct_id: %s for key: %s and value: %s", distinct_id, key, value)
	case "add_whatsapp":
		userInstance.AddWhatsapp(value)
		out = fmt.Sprintf("Whatsapp added successfully for user with distinct_id: %s for key: %s and value: %s", distinct_id, key, value)
	case "remove_whatsapp":
		userInstance.RemoveWhatsapp(value)
		out = fmt.Sprintf("Whatsapp removed successfully for user with distinct_id: %s for key: %s and value: %s", distinct_id, key, value)
	case "add_androidpush":
		userInstance.AddAndroidpush(value, identity_provider)
		out = fmt.Sprintf("Android push added successfully for user with distinct_id: %s for key: %s and value: %s and identity_provider: %s", distinct_id, key, value, identity_provider)
	case "remove_androidpush":
		userInstance.RemoveAndroidpush(value, identity_provider)
		out = fmt.Sprintf("Android push removed successfully for user with distinct_id: %s for key: %s and value: %s and identity_provider: %s", distinct_id, key, value, identity_provider)
	case "add_iospush":
		userInstance.AddIospush(value, identity_provider)
		out = fmt.Sprintf("iOS push added successfully for user with distinct_id: %s for key: %s and value: %s and identity_provider: %s", distinct_id, key, value, identity_provider)
	case "remove_iospush":
		userInstance.RemoveIospush(value, identity_provider)
		out = fmt.Sprintf("iOS push removed successfully for user with distinct_id: %s for key: %s and value: %s and identity_provider: %s", distinct_id, key, value, identity_provider)
	case "add_slack", "remove_slack":
		payload, slackOut, err := prepareSlackPayload(slack_details)
		if err != nil {
			return "", err
		}
		if action == "add_slack" {
			userInstance.AddSlack(payload)
		} else {
			userInstance.RemoveSlack(payload)
		}
		out = fmt.Sprintf(slackOut, distinct_id, value)
	}

	return out, err
}

func prepareSlackPayload(slackDetails map[string]any) (map[string]any, string, error) {
	var payload map[string]any
	var slackOut string

	if url, ok := slackDetails["slack_incoming_webhook_url"]; ok {
		payload = map[string]any{"incoming_webhook": map[string]any{"url": url}}
		slackOut = "Slack incoming webhook %s successfully for user with distinct_id: %s and value: %s"
	} else {
		accessToken, accessTokenOk := slackDetails["access_token"]
		slackEmail, slackEmailOk := slackDetails["slack_email"]
		slackChannelID, slackChannelIDOk := slackDetails["slack_channel_id"]
		slackUserID, slackUserIDOk := slackDetails["slack_user_id"]

		if slackEmailOk {
			if !accessTokenOk {
				return nil, "", errors.New("access_token is required when slack_email is provided")
			}
			payload = map[string]any{
				"access_token": accessToken,
				"email":        slackEmail,
			}
			slackOut = "Slack email %s successfully for user with distinct_id: %s and value: %s"
		} else if slackChannelIDOk {
			if !accessTokenOk {
				return nil, "", errors.New("access_token is required when slack_channel_id is provided")
			}
			payload = map[string]any{
				"access_token": accessToken,
				"channel_id":   slackChannelID,
			}
			slackOut = "Slack channel %s successfully for user with distinct_id: %s and value: %s"
		} else if slackUserIDOk {
			if !accessTokenOk {
				return nil, "", errors.New("access_token is required when slack_user_id is provided")
			}
			payload = map[string]any{
				"access_token": accessToken,
				"user_id":      slackUserID,
			}
			slackOut = "Slack user id %s successfully for user with distinct_id: %s and value: %s"
		} else {
			return nil, "", errors.New("slack_email, slack_channel_id or slack_user_id is required")
		}
	}

	return payload, slackOut, nil
}

func ToStringSlice(in []any) ([]string, error) {
	out := make([]string, len(in))
	for i, v := range in {
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("element %d is not a string (type %T)", i, v)
		}
		out[i] = s
	}
	return out, nil
}

func Remarshal(src any, dst any) error {
	data, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}
