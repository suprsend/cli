package tools

import (
	"context"
	"errors"

	"gopkg.in/yaml.v3"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/suprsend/cli/internal/utils"
	suprsend "github.com/suprsend/suprsend-go"
)

func getUserHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	distinct_id, err := request.RequireString("distinct_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	workspace := request.GetString("workspace", "staging")

	suprsend_client, err := utils.GetSuprSendWorkspaceClient(workspace)
	if err != nil {
		return nil, err
	}
	user, err := suprsend_client.Users.Get(ctx, distinct_id)
	if err != nil {
		return nil, err
	}

	yamluser, err := yaml.Marshal(user)
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(string(yamluser)), nil
}

func upsertUserHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	distinctId, err := request.RequireString("distinct_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	workspace := request.GetString("workspace", "staging")

	action, err := request.RequireString("action")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if action == "" {
		return mcp.NewToolResultError("action is required"), nil
	}

	key := request.GetString("key", "")
	value := request.GetString("value", "")

	if utils.RequiresKey(action) && key == "" {
		return mcp.NewToolResultError("key is required for " + action), nil
	}

	if utils.RequiresValue(action) && value == "" {
		return mcp.NewToolResultError("value is required for " + action), nil
	}

	slack_details, err := getSlackDetails(request, action)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ms_teams_details, err := getMSTeamsDetails(request, action)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	suprsendClient, err := utils.GetSuprSendWorkspaceClient(workspace)
	// todo:make everywhere mcp error is returned
	if err != nil {
		return nil, err
	}
	userInstance := suprsendClient.Users.GetEditInstance(distinctId)

	webpush_details, err := getWebpushDetails(request, action)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	out, err := utils.HandleUserAction(ctx, userInstance, action, key, value, slack_details, ms_teams_details, webpush_details, distinctId, workspace)
	if err != nil {
		return nil, err
	}

	_, err = suprsendClient.Users.Edit(ctx, suprsend.UserEditRequest{EditInstance: userInstance})
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(out), nil
}

func getUserPreferencesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	distinctId, err := request.RequireString("distinct_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	workspace := request.GetString("workspace", "staging")
	tenantId := request.GetString("tenant_id", "default")
	category, err := request.RequireString("category")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	channelPreferences := request.GetBool("channel_preferences", false)
	suprsendClient, err := utils.GetSuprSendWorkspaceClient(workspace)
	if err != nil {
		return nil, err
	}

	var userPref interface{}
	if category == "" {
		userPref, err = suprsendClient.Users.GetFullPreference(ctx, distinctId, &suprsend.UserFullPreferencesOptions{TenantId: tenantId})
		if err != nil {
			return nil, err
		}
	} else {
		userPref, err = suprsendClient.Users.GetCategoryPreference(ctx, distinctId, category, &suprsend.UserCategoryPreferenceOptions{TenantId: tenantId})
		if err != nil {
			return nil, err
		}
	}

	if channelPreferences {
		userPref, err = suprsendClient.Users.GetGlobalChannelsPreference(ctx, distinctId, &suprsend.UserGlobalChannelsPreferenceOptions{TenantId: tenantId})
		if err != nil {
			return nil, err
		}
	}
	yamluser, err := yaml.Marshal(userPref)
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(string(yamluser)), nil
}

func updateUserPreference(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	distinctIdsAny, ok := request.GetArguments()["distinct_ids"].([]any)
	if !ok {
		return mcp.NewToolResultError("distinct_ids must be an array"), nil
	}
	var distinctIds = []string{}
	err := utils.Remarshal(distinctIdsAny, &distinctIds)
	if err != nil {
		return mcp.NewToolResultError("distinct_ids must be an array of strings"), nil
	}

	channelPreferencesAny, ok := request.GetArguments()["channel_preferences"].([]any)
	if !ok {
		return mcp.NewToolResultError("channel_preferences must be an array"), nil
	}
	var channelPreferences = []*suprsend.UserGlobalChannelPreference{}
	err = utils.Remarshal(channelPreferencesAny, &channelPreferences)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	categoriesAny, ok := request.GetArguments()["categories"].([]any)
	if !ok {
		return mcp.NewToolResultError("categories must be an array"), nil
	}
	var categories = []*suprsend.UserCategoryPreferenceIn{}
	err = utils.Remarshal(categoriesAny, &categories)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	prefPayload := suprsend.UserBulkPreferenceUpdateBody{
		DistinctIDs:        distinctIds,
		ChannelPreferences: channelPreferences,
		Categories:         categories,
	}

	workspace := request.GetString("workspace", "staging")
	suprsendClient, err := utils.GetSuprSendWorkspaceClient(workspace)
	if err != nil {
		return nil, err
	}

	userPref, err := suprsendClient.Users.BulkUpdatePreferences(ctx, prefPayload, nil)
	if err != nil {
		return nil, err
	}

	yamlPref, err := yaml.Marshal(userPref)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(yamlPref)), nil
}

func updateUserChannelPreferenceHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	channelPreferencesAny, ok := request.GetArguments()["channel_preferences"].([]any)
	if !ok {
		return mcp.NewToolResultError("channel_preferences must be an array"), nil
	}
	var channelPreferences = []suprsend.UserGlobalChannelPreference{}
	err := utils.Remarshal(channelPreferencesAny, &channelPreferences)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	prefPayload := suprsend.UserGlobalChannelsPreferenceUpdateBody{
		ChannelPreferences: channelPreferences,
	}
	distinctId, err := request.RequireString("distinct_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	workspace := request.GetString("workspace", "staging")
	suprsendClient, err := utils.GetSuprSendWorkspaceClient(workspace)
	if err != nil {
		return nil, err
	}
	userPref, err := suprsendClient.Users.UpdateGlobalChannelsPreference(ctx, distinctId, prefPayload, nil)
	if err != nil {
		return nil, err
	}
	yamlPref, err := yaml.Marshal(userPref)
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(string(yamlPref)), nil
}

func getUserListSubscriptionsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	distinctId, err := request.RequireString("distinct_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	limit := request.GetInt("limit", 20)
	workspace := request.GetString("workspace", "staging")
	suprsendClient, err := utils.GetSuprSendWorkspaceClient(workspace)
	if err != nil {
		return nil, err
	}

	userListSubscriptions, err := suprsendClient.Users.GetListsSubscribedTo(ctx, distinctId, &suprsend.CursorListApiOptions{Limit: limit})
	if err != nil {
		return nil, err
	}

	yamlUserListSubscriptions, err := yaml.Marshal(userListSubscriptions)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(yamlUserListSubscriptions)), nil
}

func getUserObjectsSubscriptionsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	distinctId, err := request.RequireString("distinct_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	limit := request.GetInt("limit", 20)
	workspace := request.GetString("workspace", "staging")
	suprsendClient, err := utils.GetSuprSendWorkspaceClient(workspace)
	if err != nil {
		return nil, err
	}

	userObjectsSubscriptions, err := suprsendClient.Users.GetObjectsSubscribedTo(ctx, distinctId, &suprsend.CursorListApiOptions{Limit: limit})
	if err != nil {
		return nil, err
	}

	yamlUserObjectsSubscriptions, err := yaml.Marshal(userObjectsSubscriptions)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(yamlUserObjectsSubscriptions)), nil
}

func newUserTools() []*Tool {
	get_suprsend_user := &Tool{
		Name:        "users.get",
		Description: "Enables querying user information",
		MCPTool: mcp.NewTool("get_suprsend_user",
			mcp.WithDescription(`Use this tool to get all properties for a user in SuprSend. This tool will return a YAML string with all the properties of the user. At top level, it will return the distinct_id, properties (all the custom properties of the user), created_at, updated_at and an array of user channels ($email, push, $sms, $whatsapp, $slack etc.). Eeach object inside will have channel value, status and perma_status (permanent status of the identity). If the workspace is not specified. ask the user to provide it before using this tool.`),
			mcp.WithString("distinct_id",
				mcp.Description(`The distinct_id of the user to get.`),
				mcp.Required(),
			),
			mcp.WithString("workspace",
				mcp.Description(`SuprSend workspace to get the user from.`),
			),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		Handler: getUserHandler,
	}

	upsert_suprsend_user := &Tool{
		Name:        "users.upsert",
		Description: "Enables upserting user information",
		MCPTool: mcp.NewTool("upsert_suprsend_user",
			mcp.WithDescription(`Use this tool to upsert a new user or update an existing user's properties.`),
			mcp.WithString("distinct_id",
				mcp.Description(`The distinct_id of the user to get.`),
				mcp.Required(),
			),
			mcp.WithString("workspace",
				mcp.Description(`SuprSend workspace to get the user from.`),
			),
			mcp.WithString("action",
				mcp.Description(
					`The action to perform.
					use action "upsert" to create a new user or update an existing user's properties.
					use action "remove" to remove a user's properties.
					use action "set" to set a user's property, don't use this when trying to add email, add sms, add whatsapp, add androidpush, add iospush, add slack use the respective actions.
					use action "unset" to unset a user's property, don't use this when trying to remove email, remove sms, remove whatsapp, remove androidpush, remove iospush, remove slack use the respective actions.
					use action "set_once" to set a user's property once, this will only set the property if it is not already set.
					use action "append" to append a value to a user's property.
					use action "increment" to increment a user's property.
					use action "add_email" to add an email to a user.
					use action "remove_email" to remove an email from a user.
					use action "add_sms" to add an SMS to a user.
					use action "remove_sms" to remove an SMS from a user.
					use action "add_whatsapp" to add a WhatsApp to a user.
					use action "remove_whatsapp" to remove a WhatsApp from a user.
					use action "add_androidpush" to add an Android push to a user.
					use action "remove_androidpush" to remove an Android push from a user.
					use action "add_iospush" to add an iOS push to a user.
					use action "remove_iospush" to remove an iOS push from a user.
					use action "add_slack" to add a Slack to a user.
					use action "remove_slack" to remove a Slack from a user.
					use action "set_preferred_language" to set a user's preferred language.
					use action "set_timezone" to set a user's timezone.`),
				mcp.Required(),
				mcp.Enum(
					"upsert",
					"remove",
					"set",
					"set_once",
					"unset",
					"append",
					"increment",
					"add_email",
					"remove_email",
					"add_sms",
					"remove_sms",
					"add_whatsapp",
					"remove_whatsapp",
					"add_androidpush",
					"remove_androidpush",
					"set_preferred_language",
					"set_timezone",
					"add_iospush",
					"remove_iospush",
					"add_slack",
					"remove_slack",
					"add_ms_teams",
					"remove_ms_teams",
					"add_webpush",
					"remove_webpush",
				),
			),
			mcp.WithString("key",
				mcp.Description(`The key on which the action is to be performed. only required for set, append, increment, unset actions.`),
			),
			mcp.WithString("value",
				mcp.Description(`The value to needs to be added/removed/set/unset/appended/incremented.`),
			),
			mcp.WithObject("slack_details",
				mcp.Description(`This is only applicable for add_slack and remove_slack actions.`),
				mcp.Properties(slackPropertiesSchema),
			),
			mcp.WithObject("ms_teams_details",
				mcp.Description(`This is only applicable for add_ms_teams and remove_ms_teams actions.`),
				mcp.Properties(msTeamsPropertiesSchema),
			),
			mcp.WithObject("webpush_details",
				mcp.Description(`This is only applicable for add_webpush and remove_webpush actions.`),
				mcp.Properties(
					map[string]interface{}{
						"keys": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"auth":   utils.StringSchema("The auth key for the webpush"),
								"p256dh": utils.StringSchema("The p256dh key for the webpush"),
							},
							"required":             []string{"auth", "p256dh"},
							"additionalProperties": false,
						},
						"endpoint": utils.StringSchema("The endpoint for the webpush"),
					},
				),
			),
			mcp.WithDestructiveHintAnnotation(true),
		),
		Handler: upsertUserHandler,
	}
	get_suprsend_user_preferences := &Tool{
		Name:        "users.get_preferences",
		Description: "Enables querying user preferences(also within a category)",
		MCPTool: mcp.NewTool("get_suprsend_user_preferences",
			mcp.WithDescription(`Use this tool to get the preferences(also within a category) for a user in SuprSend.`),
			mcp.WithString("distinct_id",
				mcp.Description(`The distinct_id of the user to get the preferences for.`),
				mcp.Required(),
			),
			mcp.WithString("tenant_id",
				mcp.Description("The tenant_id of the tenant to get the preferences for."),
			),
			mcp.WithString("category",
				mcp.Description("The category_slug of a category to get."),
			),
			mcp.WithBoolean("channel_preferences",
				mcp.Description("Whether to include channel preferences in the response. Default is false."),
			),
			mcp.WithString("workspace",
				mcp.Description(`SuprSend workspace to get the user from.`),
			),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		Handler: getUserPreferencesHandler,
	}

	update_suprsend_users_preferences := &Tool{
		Name:        "user.update_preferences",
		Description: "Enables updating preferences for users, controlling notification preferences and channel opt-outs.",
		MCPTool: mcp.NewTool("update_suprsend_users_preferences",
			mcp.WithDescription("Use this tool to update preferences for users, controlling notification preferences and channel opt-outs."),
			mcp.WithArray("distinct_ids",
				mcp.Description("The distinct_ids of the users to update the preferences for."),
				mcp.WithStringItems(),
				mcp.Required(),
			),
			mcp.WithArray("channel_preferences",
				mcp.Description("The channel preferences to update for the users."),
				mcp.Items(map[string]any{
					"type": "object",
					"properties": map[string]any{
						"channel":       utils.StringSchema("The channel identifier"),
						"is_restricted": utils.BoolSchema("Whether the channel is restricted"),
					},
					"required": []string{"channel", "is_restricted"},
				}),
				mcp.Required(),
			),
			mcp.WithArray("categories",
				mcp.Description("The categories to update the preferences for."),
				mcp.Items(map[string]any{
					"type": "object",
					"properties": map[string]any{
						"category": utils.StringSchema("The category identifier"),
						"preference": map[string]any{
							"type":        "string",
							"description": "The preference to update for the category",
							"enum": []string{
								"opt_in",
								"opt_out",
							},
						},
						"opt_out_channels": utils.ArraySchema("The channels to opt out from for the category"),
					},
					"required": []string{"category", "preference", "opt_out_channels"},
				}),
				mcp.Required(),
			),
			mcp.WithString("workspace",
				mcp.Description("SuprSend workspace to run the query from."),
			),
			mcp.WithDestructiveHintAnnotation(true),
		),
		Handler: updateUserPreference,
	}

	update_suprsend_user_channel_preference := &Tool{
		Name:        "users.update_channel_preference",
		Description: "Enables updating channel preference for a user",
		MCPTool: mcp.NewTool("update_suprsend_user_channel_preference",
			mcp.WithDescription("Use this tool to update channel preference for a user."),
			mcp.WithString("distinct_id",
				mcp.Description("The distinct_id of the user to update the channel preference for."),
				mcp.Required(),
			),
			mcp.WithString("workspace",
				mcp.Description("SuprSend workspace to update the channel preference for."),
			),
			mcp.WithArray("channel_preferences",
				mcp.Description("The channel preferences to update for the users."),
				mcp.Items(map[string]any{
					"type": "object",
					"properties": map[string]any{
						"channel":       utils.StringSchema("The channel identifier"),
						"is_restricted": utils.BoolSchema("Whether the channel is restricted"),
					},
					"required": []string{"channel", "is_restricted"},
				}),
				mcp.Required(),
			),
			mcp.WithDestructiveHintAnnotation(true),
		),
		Handler: updateUserChannelPreferenceHandler,
	}

	get_suprsend_user_list_subscriptions := &Tool{
		Name:        "users.get_list_subscriptions",
		Description: "Enables querying list subscriptions for a user",
		MCPTool: mcp.NewTool("get_suprsend_user_list_subscriptions",
			mcp.WithDescription("Use this tool to query list subscriptions for a user."),
			mcp.WithString("distinct_id",
				mcp.Description("The distinct_id of the user to get the list subscriptions for."),
				mcp.Required(),
			),
			mcp.WithString("workspace",
				mcp.Description("SuprSend workspace to run the query from."),
			),
			mcp.WithNumber("limit",
				mcp.Description("Number of list subscriptions to get for a user."),
			),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		Handler: getUserListSubscriptionsHandler,
	}

	get_suprsend_user_objects_subscriptions := &Tool{
		Name:        "users.get_objects_subscriptions",
		Description: "Enables querying object subscriptions for a user",
		MCPTool: mcp.NewTool("get_suprsend_user_objects_subscriptions",
			mcp.WithDescription("Use this tool to query object subscriptions for a user."),
			mcp.WithString("distinct_id",
				mcp.Description("The distinct_id of the user to get the object subscriptions for."),
				mcp.Required(),
			),
			mcp.WithString("workspace",
				mcp.Description("SuprSend workspace to run the query from."),
			),
			mcp.WithNumber("limit",
				mcp.Description("Number of object subscriptions to get for a user."),
			),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		Handler: getUserObjectsSubscriptionsHandler,
	}

	tools := []*Tool{
		get_suprsend_user,
		upsert_suprsend_user,
		get_suprsend_user_preferences,
		update_suprsend_users_preferences,
		update_suprsend_user_channel_preference,
		get_suprsend_user_list_subscriptions,
		get_suprsend_user_objects_subscriptions,
	}
	return tools
}

func init() {
	for _, t := range newUserTools() {
		RegisterTool(t, "users")
	}
}

var slackPropertiesSchema = map[string]any{
	"access_token":               utils.StringSchema("Access token for the Slack workspace"),
	"slack_email":                utils.StringSchema("Email of the user to add to the Slack workspace"),
	"slack_channel_id":           utils.StringSchema("ID of the Slack channel"),
	"slack_user_id":              utils.StringSchema("ID of the Slack user"),
	"slack_incoming_webhook_url": utils.StringSchema("Incoming webhook URL for the Slack"),
}

var msTeamsPropertiesSchema = map[string]interface{}{
	"type": map[string]interface{}{
		"required": true,
		"type":     "string",
		"enum": []string{
			"incoming_webhook",
			"channel",
			"user",
			"user_id",
		},
	},
	"incoming_webhook": map[string]interface{}{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]interface{}{
			"url": map[string]interface{}{
				"type":             "string",
				"format":           "uri",
				"qt-uri-protocols": []string{"https"},
			},
		},
		"required": []string{"url"},
		"title":    "IncomingWebhook",
	},
	"channel": map[string]interface{}{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]interface{}{
			"tenant_id": map[string]interface{}{
				"type": "string",
			},
			"service_url": map[string]interface{}{
				"type":             "string",
				"format":           "uri",
				"qt-uri-protocols": []string{"https"},
			},
			"conversation_id": map[string]interface{}{
				"type": "string",
			},
		},
		"required": []string{"conversation_id", "service_url", "tenant_id"},
		"title":    "Channel",
	},
	"user": map[string]interface{}{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]interface{}{
			"tenant_id": map[string]interface{}{
				"type": "string",
			},
			"service_url": map[string]interface{}{
				"type":             "string",
				"format":           "uri",
				"qt-uri-protocols": []string{"https"},
			},
			"conversation_id": map[string]interface{}{
				"type": "string",
			},
		},
		"required": []string{"conversation_id", "service_url", "tenant_id"},
		"title":    "Channel",
	},
	"user_id": map[string]interface{}{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]interface{}{
			"tenant_id": map[string]interface{}{
				"type": "string",
			},
			"service_url": map[string]interface{}{
				"type":             "string",
				"format":           "uri",
				"qt-uri-protocols": []string{"https"},
			},
			"user_id": map[string]interface{}{
				"type": "string",
			},
		},
		"required": []string{"tenant_id", "user_id", "service_url"},
		"title":    "UserID",
	},
}

func getSlackDetails(request mcp.CallToolRequest, action string) (map[string]any, error) {
	if action != "add_slack" && action != "remove_slack" {
		return nil, nil
	}

	slackDetailsRaw, ok := request.GetArguments()["slack_details"]
	if !ok {
		return nil, errors.New("required argument 'slack_details' not found")
	}

	slackDetails, ok := slackDetailsRaw.(map[string]any)
	if !ok {
		return nil, errors.New("invalid slack_details")
	}
	return slackDetails, nil
}

func getMSTeamsDetails(request mcp.CallToolRequest, action string) (map[string]any, error) {
	if action != "add_ms_teams" && action != "remove_ms_teams" {
		return nil, nil
	}

	msTeamsDetailsRaw, ok := request.GetArguments()["ms_teams_details"]
	if !ok {
		return nil, errors.New("required argument 'ms_teams_details' not found")
	}

	msTeamsDetails, ok := msTeamsDetailsRaw.(map[string]any)
	if !ok {
		return nil, errors.New("invalid ms_teams_details")
	}

	return msTeamsDetails, nil
}

func getWebpushDetails(request mcp.CallToolRequest, action string) (map[string]any, error) {
	if action != "add_webpush" && action != "remove_webpush" {
		return nil, nil
	}

	webpushDetailsRaw, ok := request.GetArguments()["webpush_details"]
	if !ok {
		return nil, errors.New("required argument 'webpush_details' not found")
	}

	webpushDetails, ok := webpushDetailsRaw.(map[string]any)
	if !ok {
		return nil, errors.New("invalid webpush_details")
	}

	return webpushDetails, nil
}
