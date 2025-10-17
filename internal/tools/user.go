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

	key := request.GetString("key", "")
	value := request.GetString("value", "")

	if utils.RequiresKey(action) && key == "" {
		return mcp.NewToolResultError("key is required for " + action), nil
	}

	if utils.RequiresKey(action) && value == "" {
		return mcp.NewToolResultError("value is required for " + action), nil
	}

	slack_details, err := getSlackDetails(request, action)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	identityProvider := request.GetString("identity_provider", "")
	suprsendClient, err := utils.GetSuprSendWorkspaceClient(workspace)
	// todo:make everywhere mcp error is returned
	if err != nil {
		return nil, err
	}
	userInstance := suprsendClient.Users.GetEditInstance(distinctId)

	out, err := utils.HandleUserAction(ctx, userInstance, action, key, value, slack_details, identityProvider, distinctId, workspace)
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
	tenantId, err := request.RequireString("tenant_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
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
	rawPayload, ok := request.GetArguments()["payload"].(map[string]any)
	if !ok {
		return mcp.NewToolResultError("payload must be an object"), nil
	}
	distinctIds, ok := rawPayload["distinct_ids"].([]string)
	if !ok {
		return mcp.NewToolResultError("distinct_ids must be an array"), nil
	}

	channelPreferences, ok := rawPayload["channel_preferences"].([]*suprsend.UserGlobalChannelPreference)
	if !ok {
		return mcp.NewToolResultError("channel_preferences must be an array"), nil
	}

	categories, ok := rawPayload["categories"].([]*suprsend.UserCategoryPreferenceIn)
	if !ok {
		return mcp.NewToolResultError("categories must be an array"), nil
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
	rawPayload, ok := request.GetArguments()["payload"].(map[string]any)
	if !ok {
		return mcp.NewToolResultError("payload must be an object"), nil
	}
	channelPreferences, ok := rawPayload["channel_preferences"].([]suprsend.UserGlobalChannelPreference)
	if !ok {
		return mcp.NewToolResultError("channel_preferences must be an array"), nil
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
				mcp.Description(`The action to perform.`),
				mcp.Required(),
				mcp.Enum(
					"upsert",
					"remove",
					"set",
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
				),
			),
			mcp.WithString("key",
				mcp.Description(`The key on which the action is to be performed. only required for set, append, increment, unset actions.`),
			),
			mcp.WithString("value",
				mcp.Description(`The value to needs to be added/removed/set/unset/appended/incremented.`),
			),
			mcp.WithString("identity_provider",
				mcp.Description(`This is only applicable for add_androidpush, remove_androidpush, add_iospush, remove_iospush actions.`),
			),
			mcp.WithObject("slack_details",
				mcp.Description(`This is only applicable for add_slack and remove_slack actions.`),
				mcp.Properties(slackPropertiesSchema),
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
			mcp.WithObject("payload",
				mcp.Description("The preferences to update for the users."),
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
			mcp.WithObject("payload",
				mcp.Description("The channel preference to update for the user."),
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
