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
	"github.com/suprsend/cli/mgmnt"
	suprsend "github.com/suprsend/suprsend-go"
)

// DeepCopyMap returns a deep copy of a map[string]any via JSON round-trip.
func DeepCopyMap(m map[string]any) map[string]any {
	b, _ := json.Marshal(m)
	var out map[string]any
	json.Unmarshal(b, &out)
	return out
}

type WorkflowPayloadSchema struct {
	Schema string `json:"schema"`
	// version_no is optional and can be nil
	Version string `json:"version_no,omitempty"`
}

type WorkflowInfo struct {
	Slug          string
	Name          string
	Description   string
	Tags          []string
	PayloadSchema WorkflowPayloadSchema
}

// parseSelector splits a comma-separated --workflows / --events flag value into
// plain slugs/names and tag selectors. Entries prefixed with `tag:` are tag
// selectors; everything else is matched against the resource's slug or name.
//
// Examples:
//
//	"all"                            → all=true
//	"none" | ""                      → none=true
//	"welcome,order-placed"           → slugs=[welcome, order-placed]
//	"tag:onboarding,tag:transac"     → tags=[onboarding, transac]
//	"welcome,tag:onboarding"         → slugs=[welcome], tags=[onboarding]
func parseSelector(flag string) (all bool, none bool, slugs []string, tags []string) {
	flag = strings.TrimSpace(flag)
	if flag == "" || flag == "none" {
		return false, true, nil, nil
	}
	if flag == "all" {
		return true, false, nil, nil
	}
	for raw := range strings.SplitSeq(flag, ",") {
		entry := strings.TrimSpace(raw)
		if entry == "" {
			continue
		}
		if tag, ok := strings.CutPrefix(entry, "tag:"); ok {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tags = append(tags, tag)
			}
			continue
		}
		slugs = append(slugs, entry)
	}
	if len(slugs) == 0 && len(tags) == 0 {
		return false, true, nil, nil
	}
	return false, false, slugs, tags
}

func GenerateUUID() string {
	return uuid.New().String()
}

func FetchWorkflowsMcp(ctx context.Context, workspace, workflowsFlag string) []WorkflowInfo {
	all, none, slugs, tags := parseSelector(workflowsFlag)
	if none {
		return nil
	}

	mgmntClient := GetSuprSendMgmntClient()
	if mgmntClient == nil {
		return nil
	}
	return fetchWorkflowsBody(ctx, mgmntClient, workspace, all, slugs, tags)
}

// FetchWorkflowsMcpFor mirrors FetchWorkflowsMcp but uses the per-tenant mgmnt
// client from ctx (via MgmntClientFor). Returns nil if ctx has no tenant
// credentials AND the singleton SDKInstance is nil. Required for
// multi-tenant MCP servers' per-tenant dynamic tool registration — that path
// runs with SDKInstance == nil and must resolve the client from ctx instead.
func FetchWorkflowsMcpFor(ctx context.Context, workspace, workflowsFlag string) []WorkflowInfo {
	all, none, slugs, tags := parseSelector(workflowsFlag)
	if none {
		return nil
	}
	mgmntClient := MgmntClientFor(ctx)
	if mgmntClient == nil {
		return nil
	}
	return fetchWorkflowsBody(ctx, mgmntClient, workspace, all, slugs, tags)
}

// fetchWorkflowsBody contains the post-client-resolution logic shared by
// FetchWorkflowsMcp and FetchWorkflowsMcpFor. Pure code-motion extraction; the
// behavior is identical to the inlined version that lived in FetchWorkflowsMcp
// prior to the per-tenant variant being added.
func fetchWorkflowsBody(ctx context.Context, mgmntClient *mgmnt.SS_MgmntClient, workspace string, all bool, slugs, tags []string) []WorkflowInfo {
	workflowsResp, err := mgmntClient.GetWorkflows(ctx, workspace, "live")
	if err != nil {
		return nil
	}

	var result []WorkflowInfo
	for _, workflow := range workflowsResp.Results {
		workflowMap, ok := workflow.(map[string]any)
		if !ok {
			continue
		}
		info := WorkflowInfo{}
		if slug, ok := workflowMap["slug"].(string); ok {
			info.Slug = slug
		}
		if name, ok := workflowMap["name"].(string); ok {
			info.Name = name
		}
		if description, ok := workflowMap["description"].(string); ok {
			info.Description = description
		}
		if rawTags, ok := workflowMap["tags"].([]any); ok {
			for _, t := range rawTags {
				if s, ok := t.(string); ok {
					info.Tags = append(info.Tags, s)
				}
			}
		}
		if payloadSchema, ok := workflowMap["payload_schema"].(map[string]any); ok {
			info.PayloadSchema.Schema, _ = payloadSchema["schema"].(string)
			if versionNo, ok := payloadSchema["version_no"].(string); ok {
				info.PayloadSchema.Version = versionNo
			}
		}

		if matchesSelector(info.Slug, info.Name, info.Tags, all, slugs, tags) {
			result = append(result, info)
		}
	}
	return result
}

// matchesSelector returns true when a resource's slug, name, or any of its
// tags matches the parsed selector. `all` short-circuits to true.
func matchesSelector(slug, name string, resourceTags []string, all bool, slugs, tags []string) bool {
	if all {
		return true
	}
	if slug != "" && slices.Contains(slugs, slug) {
		return true
	}
	if name != "" && slices.Contains(slugs, name) {
		return true
	}
	for _, t := range resourceTags {
		if slices.Contains(tags, t) {
			return true
		}
	}
	return false
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

func FetchEventsMcp(ctx context.Context, workspace string, eventsFlag string) []EventInfo {
	// Selector accepts `none`, `all`, comma-separated names, and (for forward
	// compatibility) `tag:<tag>` entries — events don't yet expose tags from
	// the API, so tag selectors will simply match nothing today.
	all, none, names, tags := parseSelector(eventsFlag)
	if none {
		return nil
	}

	mgmntClient := GetSuprSendMgmntClient()
	if mgmntClient == nil {
		return nil
	}
	return fetchEventsBody(ctx, mgmntClient, workspace, all, names, tags)
}

// FetchEventsMcpFor mirrors FetchEventsMcp but uses the per-tenant mgmnt
// client from ctx (via MgmntClientFor). Returns nil if ctx has no tenant
// credentials AND the singleton SDKInstance is nil. Required for
// multi-tenant MCP servers' per-tenant dynamic tool registration — that path
// runs with SDKInstance == nil and must resolve the client from ctx instead.
func FetchEventsMcpFor(ctx context.Context, workspace, eventsFlag string) []EventInfo {
	all, none, names, tags := parseSelector(eventsFlag)
	if none {
		return nil
	}
	mgmntClient := MgmntClientFor(ctx)
	if mgmntClient == nil {
		return nil
	}
	return fetchEventsBody(ctx, mgmntClient, workspace, all, names, tags)
}

// fetchEventsBody contains the post-client-resolution logic shared by
// FetchEventsMcp and FetchEventsMcpFor. Pure code-motion extraction; the
// behavior is identical to the inlined version that lived in FetchEventsMcp
// prior to the per-tenant variant being added.
func fetchEventsBody(ctx context.Context, mgmntClient *mgmnt.SS_MgmntClient, workspace string, all bool, names, tags []string) []EventInfo {
	eventsResp, err := mgmntClient.GetEvents(ctx, workspace)
	if err != nil {
		return nil
	}
	var result []EventInfo
	for _, event := range eventsResp.Results {
		eventMap, ok := event.(map[string]any)
		if !ok {
			continue
		}
		info := EventInfo{}
		if name, ok := eventMap["name"].(string); ok {
			info.Name = name
		}
		if description, ok := eventMap["description"].(string); ok {
			info.Description = description
		}
		if payloadSchema, ok := eventMap["payload_schema"].(map[string]any); ok {
			info.PayloadSchema.Schema, _ = payloadSchema["schema"].(string)
			if versionNo, ok := payloadSchema["version_no"].(string); ok {
				info.PayloadSchema.Version = versionNo
			}
		}

		if matchesSelector(info.Name, info.Name, nil, all, names, tags) {
			result = append(result, info)
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

func RequiresValue(action string) bool {
	actions := map[string]bool{
		"set": true, "append": true, "increment": true,
	}
	return actions[action]
}

func HandleObjectAction(ctx context.Context, objectInstance suprsend.ObjectEdit, action, key, value string, slack_details map[string]any, ms_teams_details map[string]any, webpush_details map[string]any, objectIdentifier suprsend.ObjectIdentifier, workspace string) (string, error) {
	var err error
	var out string

	suprsend_client, err := GetSuprSendWorkspaceClient(workspace, ctx)
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
	case "set_once":
		objectInstance.SetOnce(map[string]any{key: value})
		out = fmt.Sprintf("Key set once successfully for user with object_id: %s for key: %s and value: %s", objectIdentifier.Id, key, value)
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
		objectInstance.AddAndroidpush(value, "")
		out = fmt.Sprintf("Android push added successfully for user with object_id: %s for key: %s and value: %s ", objectIdentifier.Id, key, value)
	case "remove_androidpush":
		objectInstance.RemoveAndroidpush(value, "")
		out = fmt.Sprintf("Android push removed successfully for user with object_id: %s for key: %s and value: %s ", objectIdentifier.Id, key, value)
	case "add_iospush":
		objectInstance.AddIospush(value, "")
		out = fmt.Sprintf("iOS push added successfully for user with object_id: %s for key: %s and value: %s ", objectIdentifier.Id, key, value)
	case "remove_iospush":
		objectInstance.RemoveIospush(value, "")
		out = fmt.Sprintf("iOS push removed successfully for user with object_id: %s for key: %s and value: %s ", objectIdentifier.Id, key, value)
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
	case "add_ms_teams", "remove_ms_teams":
		payload, msTeamsOut, err := prepareMSTeamsPayload(ms_teams_details)
		if err != nil {
			return "", err
		}
		if action == "add_ms_teams" {
			objectInstance.AddMSTeams(payload)
		} else {
			objectInstance.RemoveMSTeams(payload)
		}
		out = fmt.Sprintf(msTeamsOut, objectIdentifier.Id, value)
	case "add_webpush", "remove_webpush":
		payload, webpushOut, err := prepareWebpushPayload(webpush_details)
		if err != nil {
			return "", err
		}
		if action == "add_webpush" {
			objectInstance.AddWebpush(payload, "vapid")
		} else {
			objectInstance.RemoveWebpush(payload, "vapid")
		}
		out = fmt.Sprintf(webpushOut, objectIdentifier.Id, value)
	}

	return out, err
}

func HandleUserAction(ctx context.Context, userInstance suprsend.UserEdit, action, key, value string, slack_details map[string]any, ms_teams_details map[string]any, webpush_details map[string]any, distinct_id string, workspace string) (string, error) {
	var err error
	var out string

	suprsend_client, err := GetSuprSendWorkspaceClient(workspace, ctx)
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
	case "set_once":
		userInstance.SetOnce(map[string]any{key: value})
		out = fmt.Sprintf("Key set once successfully for user with distinct_id: %s for key: %s and value: %s", distinct_id, key, value)
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
		userInstance.AddAndroidpush(value, "")
		out = fmt.Sprintf("Android push added successfully for user with distinct_id: %s for key: %s and value: %s ", distinct_id, key, value)
	case "remove_androidpush":
		userInstance.RemoveAndroidpush(value, "")
		out = fmt.Sprintf("Android push removed successfully for user with distinct_id: %s for key: %s and value: %s ", distinct_id, key, value)
	case "add_iospush":
		userInstance.AddIospush(value, "")
		out = fmt.Sprintf("iOS push added successfully for user with distinct_id: %s for key: %s and value: %s ", distinct_id, key, value)
	case "remove_iospush":
		userInstance.RemoveIospush(value, "")
		out = fmt.Sprintf("iOS push removed successfully for user with distinct_id: %s for key: %s and value: %s ", distinct_id, key, value)
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
	case "add_ms_teams", "remove_ms_teams":
		payload, msTeamsOut, err := prepareMSTeamsPayload(ms_teams_details)
		if err != nil {
			return "", err
		}
		if action == "add_ms_teams" {
			userInstance.AddMSTeams(payload)
		} else {
			userInstance.RemoveMSTeams(payload)
		}
		out = fmt.Sprintf(msTeamsOut, distinct_id, value)
	case "add_webpush", "remove_webpush":
		payload, webpushOut, err := prepareWebpushPayload(webpush_details)
		if err != nil {
			return "", err
		}
		if action == "add_webpush" {
			userInstance.AddWebpush(payload, "vapid")
		} else {
			userInstance.RemoveWebpush(payload, "vapid")
		}
		out = fmt.Sprintf(webpushOut, distinct_id, value)
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

func prepareMSTeamsPayload(msTeamsDetails map[string]any) (map[string]any, string, error) {
	msteamsType, ok := msTeamsDetails["type"].(string)
	if !ok || msteamsType == "" {
		return nil, "", errors.New("type is required for ms_teams_details and must be one of: incoming_webhook, channel, user, user_id")
	}
	var payload map[string]any
	var msTeamsOut string
	switch msteamsType {
	case "incoming_webhook":
		incomingWebhook, ok := msTeamsDetails["incoming_webhook"].(map[string]any)
		if !ok {
			return nil, "", errors.New("incoming_webhook is required for ms_teams_details when type is incoming_webhook")
		}
		url, ok := incomingWebhook["url"].(string)
		if !ok || url == "" {
			return nil, "", errors.New("url is required for incoming_webhook")
		}
		payload = map[string]any{"incoming_webhook": map[string]any{"url": url}}
		msTeamsOut = "Incoming webhook added successfully for user with distinct_id: %s and value: %s"
	case "channel":
		channel, ok := msTeamsDetails["channel"].(map[string]any)
		if !ok {
			return nil, "", errors.New("channel is required for ms_teams_details when type is channel")
		}
		tenantId, ok := channel["tenant_id"].(string)
		if !ok || tenantId == "" {
			return nil, "", errors.New("tenant_id is required for channel")
		}
		serviceUrl, ok := channel["service_url"].(string)
		if !ok || serviceUrl == "" {
			return nil, "", errors.New("service_url is required for channel")
		}
		conversationId, ok := channel["conversation_id"].(string)
		if !ok || conversationId == "" {
			return nil, "", errors.New("conversation_id is required for channel")
		}
		payload = map[string]any{"tenant_id": tenantId, "service_url": serviceUrl, "conversation_id": conversationId}
		msTeamsOut = "Channel added successfully for user with distinct_id: %s and value: %s"
	case "user":
		user, ok := msTeamsDetails["user"].(map[string]any)
		if !ok {
			return nil, "", errors.New("user is required for ms_teams_details when type is user")
		}
		tenantId, ok := user["tenant_id"].(string)
		if !ok || tenantId == "" {
			return nil, "", errors.New("tenant_id is required for user")
		}
		serviceUrl, ok := user["service_url"].(string)
		if !ok || serviceUrl == "" {
			return nil, "", errors.New("service_url is required for user")
		}
		conversationId, ok := user["conversation_id"].(string)
		if !ok || conversationId == "" {
			return nil, "", errors.New("conversation_id is required for user")
		}
		payload = map[string]any{"tenant_id": tenantId, "service_url": serviceUrl, "conversation_id": conversationId}
		msTeamsOut = "User added successfully for user with distinct_id: %s and value: %s"
	case "user_id":
		userId, ok := msTeamsDetails["user_id"].(map[string]any)
		if !ok {
			return nil, "", errors.New("user_id is required for ms_teams_details when type is user_id")
		}
		userIdValue, ok := userId["user_id"].(string)
		if !ok || userIdValue == "" {
			return nil, "", errors.New("user_id is required for user_id")
		}
		tenantId, ok := userId["tenant_id"].(string)
		if !ok || tenantId == "" {
			return nil, "", errors.New("tenant_id is required for user_id")
		}
		serviceUrl, ok := userId["service_url"].(string)
		if !ok || serviceUrl == "" {
			return nil, "", errors.New("service_url is required for user_id")
		}
		payload = map[string]any{"user_id": userIdValue, "tenant_id": tenantId, "service_url": serviceUrl}
	default:
		return nil, "", errors.New("invalid type for ms_teams_details")
	}
	return payload, msTeamsOut, nil
}

func prepareWebpushPayload(webpushDetails map[string]any) (map[string]any, string, error) {
	var payload map[string]any
	var webpushOut string

	keys, ok := webpushDetails["keys"].(map[string]any)
	if !ok {
		return nil, "", errors.New("keys is required for webpush_details")
	}
	auth, ok := keys["auth"].(string)
	if !ok {
		return nil, "", errors.New("auth is required for webpush_details")
	}
	p256dh, ok := keys["p256dh"].(string)
	if !ok {
		return nil, "", errors.New("p256dh is required for webpush_details")
	}
	endpoint, ok := webpushDetails["endpoint"].(string)
	if !ok || endpoint == "" {
		return nil, "", errors.New("endpoint is required for webpush_details")
	}
	payload = map[string]any{"keys": map[string]any{"auth": auth, "p256dh": p256dh}, "endpoint": endpoint}
	webpushOut = "Webpush added successfully for user with distinct_id: %s and value: %s"
	return payload, webpushOut, nil
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
