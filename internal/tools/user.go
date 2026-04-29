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
		MCPTool: mcp.NewTool("get_suprsend_user",
			mcp.WithDescription(`Get a SuprSend user's full state by distinct_id. Users are end recipients of notifications, identified by your application's user id.

When to use: the user references a recipient by id and you need their stored properties or channel identifiers.

When NOT to use:
- For non-user entities (organizations, projects, vehicles) — use get_suprsend_object.
- For preferences only — use get_suprsend_user_preferences.
- For mailing-list / object subscriptions — use get_suprsend_user_list_subscriptions or get_suprsend_user_objects_subscriptions.

Returns: YAML with distinct_id, properties (custom fields like name, plan, lang), created_at, updated_at, and a channels array — each entry has the channel value, status, and perma_status (e.g., bounced, blocked, soft-bounced).`),
			mcp.WithString("distinct_id",
				mcp.Description(`The distinct_id of the user to get.`),
				mcp.Required(),
			),
			mcp.WithString("workspace",
				mcp.Description(`SuprSend workspace to get the user from.`),
			),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		Handler: getUserHandler,
	}

	upsert_suprsend_user := &Tool{
		Name:        "users.upsert",
		MCPTool: mcp.NewTool("upsert_suprsend_user",
			mcp.WithDescription(`Modify properties or channel identifiers on a SuprSend user. One call performs ONE action; for multiple changes, call this tool multiple times.

Actions:
- set, set_once, unset, remove — modify a scalar property by key/value. remove permanently deletes the key.
- append, increment — modify array / numeric values.
- set_preferred_language, set_timezone — locale and timezone on the user.
- add_email / remove_email, add_sms / remove_sms, add_whatsapp / remove_whatsapp, add_androidpush / remove_androidpush, add_iospush / remove_iospush, add_slack / remove_slack — register or deregister a delivery channel.

Channel registration is special. For channel identifiers ALWAYS use the dedicated add_<channel> / remove_<channel> actions — generic set / unset will not register the channel correctly with the delivery router. Slack additionally requires the slack_details payload alongside the action.

When to use: creating a new user or modifying an existing user's stored state.

When NOT to use:
- For non-user entities — use upsert_suprsend_object.
- For preferences — use update_suprsend_users_preferences (per category) or update_suprsend_user_channel_preference (across categories).
- For tenant settings — use upsert_suprsend_tenant.

Side effects: remove and unset permanently delete data. add_<channel> makes the user reachable on that channel for any future workflow run; remove_<channel> stops delivery immediately.

Returns: the updated user on success; structured error with field reasons on failure.`),
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
				msTeamsRequiredFields(),
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
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		Handler: upsertUserHandler,
	}
	get_suprsend_user_preferences := &Tool{
		Name:        "users.get_preferences",
		MCPTool: mcp.NewTool("get_suprsend_user_preferences",
			mcp.WithDescription(`Read a user's category-level notification preferences and (optionally) per-channel overrides.

When to use:
- Before update_suprsend_users_preferences, to read current state.
- The user asks what categories a recipient is opted in/out of.
- Before sending, to check delivery permission for a category or channel.

When NOT to use:
- For the user's identity or channel identifiers — use get_suprsend_user.
- For tenant-level defaults — use get_tenant_default_preference.
- For object preferences — use get_suprsend_object_preferences.

Returns: the user's preference tree. Pass category to scope to one preference; omit for the full tree. Set channel_preferences=true to include per-channel overrides.`),
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
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		Handler: getUserPreferencesHandler,
	}

	update_suprsend_users_preferences := &Tool{
		Name:        "user.update_preferences",
		MCPTool: mcp.NewTool("update_suprsend_users_preferences",
			mcp.WithDescription(`Set ONE category's preference for ONE user — opted in, opted out, or cant_unsubscribe (locked) — plus per-channel opt-outs within that category.

Replaces, does not merge. This call overwrites the existing preference for the named category. Previous opt-outs within the same category are lost; pass them again in opt_out_channels if you want to keep them.

When to use: changing a single category for a single user.

When NOT to use:
- For tenant-wide defaults — use update_suprsend_tenant_default_preference.
- For cross-category channel toggles ("block all SMS") — use update_suprsend_user_channel_preference.
- For the same flow on objects — use update_suprsend_category_preference_object.

Preference values: opt_in enables; opt_out disables; cant_unsubscribe locks the user from toggling this category in their preference UI.

Returns: updated preference state on success; structured error on failure (e.g., unknown category slug).`),
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
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		Handler: updateUserPreference,
	}

	update_suprsend_user_channel_preference := &Tool{
		Name:        "users.update_channel_preference",
		MCPTool: mcp.NewTool("update_suprsend_user_channel_preference",
			mcp.WithDescription(`Block or allow specific delivery channels for ONE user, applied across ALL categories. Use this for "block all SMS to this user" or "allow only email" patterns.

is_restricted semantics: true blocks delivery on that channel; false re-enables it. Each entry in channel_preferences is a {channel, is_restricted} pair.

When NOT to use:
- For per-category control — use update_suprsend_users_preferences.
- For permanent invalidation (e.g., a bounced email) — that's set by SuprSend automatically as perma_status; don't try to override it here.
- For objects — use update_suprsend_object_channel_preference.

Side effects: takes effect on the next workflow run; in-flight notifications already in the queue may still send.

Returns: updated channel-preference state on success.`),
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
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		Handler: updateUserChannelPreferenceHandler,
	}

	get_suprsend_user_list_subscriptions := &Tool{
		Name:        "users.get_list_subscriptions",
		MCPTool: mcp.NewTool("get_suprsend_user_list_subscriptions",
			mcp.WithDescription(`List the SuprSend Lists this user belongs to. Lists are workspace-level recipient groups (segments / mailing lists), distinct from object follows.

When to use: the user asks "what mailing lists is X on?" or "what segments include X?".

When NOT to use:
- For object follows (X follows project Y) — use get_suprsend_user_objects_subscriptions.
- For followers OF an object — use get_suprsend_object_subscriptions.

Returns: a paginated list of List metadata. Default limit is 20; raise it for larger results.`),
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
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		Handler: getUserListSubscriptionsHandler,
	}

	get_suprsend_user_objects_subscriptions := &Tool{
		Name:        "users.get_objects_subscriptions",
		MCPTool: mcp.NewTool("get_suprsend_user_objects_subscriptions",
			mcp.WithDescription(`List the objects this user is subscribed TO — what the user follows.

When to use: the user asks "what does X follow?", "what projects is X in?", or you need to enumerate a user's outbound subscriptions.

When NOT to use:
- For followers OF an object (inverse direction) — use get_suprsend_object_subscriptions.
- For mailing-list / segment membership — use get_suprsend_user_list_subscriptions.

Returns: a paginated list of {object_type, object_id} entries. Default limit is 20.`),
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
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
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
		"type": "string",
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
				"type":   "string",
				"format": "uri",
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
				"type":   "string",
				"format": "uri",
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
				"type":   "string",
				"format": "uri",
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
				"type":   "string",
				"format": "uri",
			},
			"user_id": map[string]interface{}{
				"type": "string",
			},
		},
		"required": []string{"tenant_id", "user_id", "service_url"},
		"title":    "UserID",
	},
}

// msTeamsRequiredFields sets the required fields on the ms_teams_details object schema.
func msTeamsRequiredFields() mcp.PropertyOption {
	return func(schema map[string]any) {
		schema["required"] = []string{"type"}
	}
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
