package tools

import (
	"context"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/cli/pkg/mcpsdk"
	suprsend "github.com/suprsend/suprsend-go"
	"gopkg.in/yaml.v3"
)

func getUserHandler(ctx context.Context, args mcpsdk.Args) (mcpsdk.Result, error) {
	distinct_id, err := args.RequireString("distinct_id")
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	workspace := args.GetString("workspace", "staging")

	suprsend_client, err := utils.GetSuprSendWorkspaceClient(workspace, ctx)
	if err != nil {
		if utils.IsAuthError(err) {
			markSessionDead(ctx)
		}
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}
	user, err := suprsend_client.Users.Get(ctx, distinct_id)
	if err != nil {
		if utils.IsAuthError(err) {
			markSessionDead(ctx)
		}
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	yamluser, err := yaml.Marshal(user)
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}
	return mcpsdk.Result{Text: string(yamluser)}, nil
}

func upsertUserHandler(ctx context.Context, args mcpsdk.Args) (mcpsdk.Result, error) {
	distinctId, err := args.RequireString("distinct_id")
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	workspace := args.GetString("workspace", "staging")

	action, err := args.RequireString("action")
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}
	if action == "" {
		return mcpsdk.Result{Text: "action is required", IsError: true}, nil
	}

	key := args.GetString("key", "")
	value := args.GetString("value", "")

	if utils.RequiresKey(action) && key == "" {
		return mcpsdk.Result{Text: "key is required for " + action, IsError: true}, nil
	}

	if utils.RequiresValue(action) && value == "" {
		return mcpsdk.Result{Text: "value is required for " + action, IsError: true}, nil
	}

	slack_details, err := getSlackDetailsFromArgs(args, action)
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	ms_teams_details, err := getMSTeamsDetailsFromArgs(args, action)
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	suprsendClient, err := utils.GetSuprSendWorkspaceClient(workspace, ctx)
	if err != nil {
		if utils.IsAuthError(err) {
			markSessionDead(ctx)
		}
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}
	userInstance := suprsendClient.Users.GetEditInstance(distinctId)

	webpush_details, err := getWebpushDetailsFromArgs(args, action)
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	out, err := utils.HandleUserAction(ctx, userInstance, action, key, value, slack_details, ms_teams_details, webpush_details, distinctId, workspace)
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	_, err = suprsendClient.Users.Edit(ctx, suprsend.UserEditRequest{EditInstance: userInstance})
	if err != nil {
		if utils.IsAuthError(err) {
			markSessionDead(ctx)
		}
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}
	return mcpsdk.Result{Text: out}, nil
}

func getUserPreferencesHandler(ctx context.Context, args mcpsdk.Args) (mcpsdk.Result, error) {
	distinctId, err := args.RequireString("distinct_id")
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}
	workspace := args.GetString("workspace", "staging")
	tenantId := args.GetString("tenant_id", "default")
	category, err := args.RequireString("category")
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}
	channelPreferences := args.GetBool("channel_preferences", false)
	suprsendClient, err := utils.GetSuprSendWorkspaceClient(workspace, ctx)
	if err != nil {
		if utils.IsAuthError(err) {
			markSessionDead(ctx)
		}
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	var userPref interface{}
	if category == "" {
		userPref, err = suprsendClient.Users.GetFullPreference(ctx, distinctId, &suprsend.UserFullPreferencesOptions{TenantId: tenantId})
		if err != nil {
			if utils.IsAuthError(err) {
				markSessionDead(ctx)
			}
			return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
		}
	} else {
		userPref, err = suprsendClient.Users.GetCategoryPreference(ctx, distinctId, category, &suprsend.UserCategoryPreferenceOptions{TenantId: tenantId})
		if err != nil {
			if utils.IsAuthError(err) {
				markSessionDead(ctx)
			}
			return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
		}
	}

	if channelPreferences {
		userPref, err = suprsendClient.Users.GetGlobalChannelsPreference(ctx, distinctId, &suprsend.UserGlobalChannelsPreferenceOptions{TenantId: tenantId})
		if err != nil {
			if utils.IsAuthError(err) {
				markSessionDead(ctx)
			}
			return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
		}
	}
	yamluser, err := yaml.Marshal(userPref)
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}
	return mcpsdk.Result{Text: string(yamluser)}, nil
}

func updateUserPreference(ctx context.Context, args mcpsdk.Args) (mcpsdk.Result, error) {
	rawArgs := args.Map()

	distinctIdsAny, ok := rawArgs["distinct_ids"].([]any)
	if !ok {
		return mcpsdk.Result{Text: "distinct_ids must be an array", IsError: true}, nil
	}
	var distinctIds = []string{}
	err := utils.Remarshal(distinctIdsAny, &distinctIds)
	if err != nil {
		return mcpsdk.Result{Text: "distinct_ids must be an array of strings", IsError: true}, nil
	}

	channelPreferencesAny, ok := rawArgs["channel_preferences"].([]any)
	if !ok {
		return mcpsdk.Result{Text: "channel_preferences must be an array", IsError: true}, nil
	}
	var channelPreferences = []*suprsend.UserGlobalChannelPreference{}
	err = utils.Remarshal(channelPreferencesAny, &channelPreferences)
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	categoriesAny, ok := rawArgs["categories"].([]any)
	if !ok {
		return mcpsdk.Result{Text: "categories must be an array", IsError: true}, nil
	}
	var categories = []*suprsend.UserCategoryPreferenceIn{}
	err = utils.Remarshal(categoriesAny, &categories)
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	prefPayload := suprsend.UserBulkPreferenceUpdateBody{
		DistinctIDs:        distinctIds,
		ChannelPreferences: channelPreferences,
		Categories:         categories,
	}

	workspace := args.GetString("workspace", "staging")
	suprsendClient, err := utils.GetSuprSendWorkspaceClient(workspace, ctx)
	if err != nil {
		if utils.IsAuthError(err) {
			markSessionDead(ctx)
		}
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	userPref, err := suprsendClient.Users.BulkUpdatePreferences(ctx, prefPayload, nil)
	if err != nil {
		if utils.IsAuthError(err) {
			markSessionDead(ctx)
		}
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	yamlPref, err := yaml.Marshal(userPref)
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	return mcpsdk.Result{Text: string(yamlPref)}, nil
}

func updateUserChannelPreferenceHandler(ctx context.Context, args mcpsdk.Args) (mcpsdk.Result, error) {
	channelPreferencesAny, ok := args.Map()["channel_preferences"].([]any)
	if !ok {
		return mcpsdk.Result{Text: "channel_preferences must be an array", IsError: true}, nil
	}
	var channelPreferences = []suprsend.UserGlobalChannelPreference{}
	err := utils.Remarshal(channelPreferencesAny, &channelPreferences)
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}
	prefPayload := suprsend.UserGlobalChannelsPreferenceUpdateBody{
		ChannelPreferences: channelPreferences,
	}
	distinctId, err := args.RequireString("distinct_id")
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}
	workspace := args.GetString("workspace", "staging")
	suprsendClient, err := utils.GetSuprSendWorkspaceClient(workspace, ctx)
	if err != nil {
		if utils.IsAuthError(err) {
			markSessionDead(ctx)
		}
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}
	userPref, err := suprsendClient.Users.UpdateGlobalChannelsPreference(ctx, distinctId, prefPayload, nil)
	if err != nil {
		if utils.IsAuthError(err) {
			markSessionDead(ctx)
		}
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}
	yamlPref, err := yaml.Marshal(userPref)
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}
	return mcpsdk.Result{Text: string(yamlPref)}, nil
}

func getUserListSubscriptionsHandler(ctx context.Context, args mcpsdk.Args) (mcpsdk.Result, error) {
	distinctId, err := args.RequireString("distinct_id")
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}
	limit := args.GetInt("limit", 20)
	workspace := args.GetString("workspace", "staging")
	suprsendClient, err := utils.GetSuprSendWorkspaceClient(workspace, ctx)
	if err != nil {
		if utils.IsAuthError(err) {
			markSessionDead(ctx)
		}
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	userListSubscriptions, err := suprsendClient.Users.GetListsSubscribedTo(ctx, distinctId, &suprsend.CursorListApiOptions{Limit: limit})
	if err != nil {
		if utils.IsAuthError(err) {
			markSessionDead(ctx)
		}
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	yamlUserListSubscriptions, err := yaml.Marshal(userListSubscriptions)
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	return mcpsdk.Result{Text: string(yamlUserListSubscriptions)}, nil
}

func getUserObjectsSubscriptionsHandler(ctx context.Context, args mcpsdk.Args) (mcpsdk.Result, error) {
	distinctId, err := args.RequireString("distinct_id")
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}
	limit := args.GetInt("limit", 20)
	workspace := args.GetString("workspace", "staging")
	suprsendClient, err := utils.GetSuprSendWorkspaceClient(workspace, ctx)
	if err != nil {
		if utils.IsAuthError(err) {
			markSessionDead(ctx)
		}
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	userObjectsSubscriptions, err := suprsendClient.Users.GetObjectsSubscribedTo(ctx, distinctId, &suprsend.CursorListApiOptions{Limit: limit})
	if err != nil {
		if utils.IsAuthError(err) {
			markSessionDead(ctx)
		}
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	yamlUserObjectsSubscriptions, err := yaml.Marshal(userObjectsSubscriptions)
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	return mcpsdk.Result{Text: string(yamlUserObjectsSubscriptions)}, nil
}

func newUserTools() []*Tool {
	get_suprsend_user := &Tool{
		Tool: &mcpsdk.Tool{
			Name: "get_suprsend_user",
			Description: `Get a SuprSend user's full state by distinct_id. Users are end recipients of notifications, identified by your application's user id.

When to use: the user references a recipient by id and you need their stored properties or channel identifiers.

When NOT to use:
- For non-user entities (organizations, projects, vehicles) — use get_suprsend_object.
- For preferences only — use get_suprsend_user_preferences.
- For mailing-list / object subscriptions — use get_suprsend_user_list_subscriptions or get_suprsend_user_objects_subscriptions.

Returns: YAML with distinct_id, properties (custom fields like name, plan, lang), created_at, updated_at, and a channels array — each entry has the channel value, status, and perma_status (e.g., bounced, blocked, soft-bounced).`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"distinct_id": {Type: "string", Description: "The distinct_id of the user to get."},
					"workspace":   {Type: "string", Description: "SuprSend workspace to get the user from."},
				},
				Required: []string{"distinct_id"},
			},
			Annotations: mcpsdk.Annotations{
				ReadOnlyHint:   true,
				IdempotentHint: true,
				OpenWorldHint:  mcpsdk.BoolPtr(true),
			},
			Handler: getUserHandler,
		},
	}

	upsert_suprsend_user := &Tool{
		Tool: &mcpsdk.Tool{
			Name: "upsert_suprsend_user",
			Description: `Modify properties or channel identifiers on a SuprSend user. One call performs ONE action; for multiple changes, call this tool multiple times.

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

Returns: the updated user on success; structured error with field reasons on failure.`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"distinct_id": {Type: "string", Description: "The distinct_id of the user to get."},
					"workspace":   {Type: "string", Description: "SuprSend workspace to get the user from."},
					"action": {
						Type: "string",
						Description: `The action to perform.
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
use action "set_timezone" to set a user's timezone.`,
						Enum: []any{
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
						},
					},
					"key":              {Type: "string", Description: "The key on which the action is to be performed. only required for set, append, increment, unset actions."},
					"value":            {Type: "string", Description: "The value to needs to be added/removed/set/unset/appended/incremented."},
					"slack_details":    slackDetailsSchema,
					"ms_teams_details": msTeamsDetailsSchema,
					"webpush_details":  webpushDetailsSchema,
				},
				Required: []string{"distinct_id", "action"},
			},
			Annotations: mcpsdk.Annotations{
				DestructiveHint: mcpsdk.BoolPtr(true),
				IdempotentHint:  false,
				OpenWorldHint:   mcpsdk.BoolPtr(true),
			},
			Handler: upsertUserHandler,
		},
	}

	get_suprsend_user_preferences := &Tool{
		Tool: &mcpsdk.Tool{
			Name: "get_suprsend_user_preferences",
			Description: `Read a user's category-level notification preferences and (optionally) per-channel overrides.

When to use:
- Before update_suprsend_users_preferences, to read current state.
- The user asks what categories a recipient is opted in/out of.
- Before sending, to check delivery permission for a category or channel.

When NOT to use:
- For the user's identity or channel identifiers — use get_suprsend_user.
- For tenant-level defaults — use get_tenant_default_preference.
- For object preferences — use get_suprsend_object_preferences.

Returns: the user's preference tree. Pass category to scope to one preference; omit for the full tree. Set channel_preferences=true to include per-channel overrides.`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"distinct_id":         {Type: "string", Description: "The distinct_id of the user to get the preferences for."},
					"tenant_id":           {Type: "string", Description: "The tenant_id of the tenant to get the preferences for."},
					"category":            {Type: "string", Description: "The category_slug of a category to get."},
					"channel_preferences": {Type: "boolean", Description: "Whether to include channel preferences in the response. Default is false."},
					"workspace":           {Type: "string", Description: "SuprSend workspace to get the user from."},
				},
				Required: []string{"distinct_id"},
			},
			Annotations: mcpsdk.Annotations{
				ReadOnlyHint:   true,
				IdempotentHint: true,
				OpenWorldHint:  mcpsdk.BoolPtr(true),
			},
			Handler: getUserPreferencesHandler,
		},
	}

	update_suprsend_users_preferences := &Tool{
		Tool: &mcpsdk.Tool{
			Name: "update_suprsend_users_preferences",
			Description: `Set ONE category's preference for ONE user — opted in, opted out, or cant_unsubscribe (locked) — plus per-channel opt-outs within that category.

Replaces, does not merge. This call overwrites the existing preference for the named category. Previous opt-outs within the same category are lost; pass them again in opt_out_channels if you want to keep them.

When to use: changing a single category for a single user.

When NOT to use:
- For tenant-wide defaults — use update_suprsend_tenant_default_preference.
- For cross-category channel toggles ("block all SMS") — use update_suprsend_user_channel_preference.
- For the same flow on objects — use update_suprsend_category_preference_object.

Preference values: opt_in enables; opt_out disables; cant_unsubscribe locks the user from toggling this category in their preference UI.

Returns: updated preference state on success; structured error on failure (e.g., unknown category slug).`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"distinct_ids": {
						Type:        "array",
						Description: "The distinct_ids of the users to update the preferences for.",
						Items:       &jsonschema.Schema{Type: "string"},
					},
					"channel_preferences": {
						Type:        "array",
						Description: "The channel preferences to update for the users.",
						Items: &jsonschema.Schema{
							Type: "object",
							Properties: map[string]*jsonschema.Schema{
								"channel":       {Type: "string", Description: "The channel identifier"},
								"is_restricted": {Type: "boolean", Description: "Whether the channel is restricted"},
							},
							Required: []string{"channel", "is_restricted"},
						},
					},
					"categories": {
						Type:        "array",
						Description: "The categories to update the preferences for.",
						Items: &jsonschema.Schema{
							Type: "object",
							Properties: map[string]*jsonschema.Schema{
								"category": {Type: "string", Description: "The category identifier"},
								"preference": {
									Type:        "string",
									Description: "The preference to update for the category",
									Enum:        []any{"opt_in", "opt_out"},
								},
								"opt_out_channels": {
									Type:        "array",
									Description: "The channels to opt out from for the category",
									Items:       &jsonschema.Schema{Type: "string"},
								},
							},
							Required: []string{"category", "preference", "opt_out_channels"},
						},
					},
					"workspace": {Type: "string", Description: "SuprSend workspace to run the query from."},
				},
				Required: []string{"distinct_ids", "channel_preferences", "categories"},
			},
			Annotations: mcpsdk.Annotations{
				DestructiveHint: mcpsdk.BoolPtr(true),
				IdempotentHint:  true,
				OpenWorldHint:   mcpsdk.BoolPtr(true),
			},
			Handler: updateUserPreference,
		},
	}

	update_suprsend_user_channel_preference := &Tool{
		Tool: &mcpsdk.Tool{
			Name: "update_suprsend_user_channel_preference",
			Description: `Block or allow specific delivery channels for ONE user, applied across ALL categories. Use this for "block all SMS to this user" or "allow only email" patterns.

is_restricted semantics: true blocks delivery on that channel; false re-enables it. Each entry in channel_preferences is a {channel, is_restricted} pair.

When NOT to use:
- For per-category control — use update_suprsend_users_preferences.
- For permanent invalidation (e.g., a bounced email) — that's set by SuprSend automatically as perma_status; don't try to override it here.
- For objects — use update_suprsend_object_channel_preference.

Side effects: takes effect on the next workflow run; in-flight notifications already in the queue may still send.

Returns: updated channel-preference state on success.`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"distinct_id": {Type: "string", Description: "The distinct_id of the user to update the channel preference for."},
					"workspace":   {Type: "string", Description: "SuprSend workspace to update the channel preference for."},
					"channel_preferences": {
						Type:        "array",
						Description: "The channel preferences to update for the users.",
						Items: &jsonschema.Schema{
							Type: "object",
							Properties: map[string]*jsonschema.Schema{
								"channel":       {Type: "string", Description: "The channel identifier"},
								"is_restricted": {Type: "boolean", Description: "Whether the channel is restricted"},
							},
							Required: []string{"channel", "is_restricted"},
						},
					},
				},
				Required: []string{"distinct_id", "channel_preferences"},
			},
			Annotations: mcpsdk.Annotations{
				DestructiveHint: mcpsdk.BoolPtr(true),
				IdempotentHint:  true,
				OpenWorldHint:   mcpsdk.BoolPtr(true),
			},
			Handler: updateUserChannelPreferenceHandler,
		},
	}

	get_suprsend_user_list_subscriptions := &Tool{
		Tool: &mcpsdk.Tool{
			Name: "get_suprsend_user_list_subscriptions",
			Description: `List the SuprSend Lists this user belongs to. Lists are workspace-level recipient groups (segments / mailing lists), distinct from object follows.

When to use: the user asks "what mailing lists is X on?" or "what segments include X?".

When NOT to use:
- For object follows (X follows project Y) — use get_suprsend_user_objects_subscriptions.
- For followers OF an object — use get_suprsend_object_subscriptions.

Returns: a paginated list of List metadata. Default limit is 20; raise it for larger results.`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"distinct_id": {Type: "string", Description: "The distinct_id of the user to get the list subscriptions for."},
					"workspace":   {Type: "string", Description: "SuprSend workspace to run the query from."},
					"limit":       {Type: "number", Description: "Number of list subscriptions to get for a user."},
				},
				Required: []string{"distinct_id"},
			},
			Annotations: mcpsdk.Annotations{
				ReadOnlyHint:   true,
				IdempotentHint: true,
				OpenWorldHint:  mcpsdk.BoolPtr(true),
			},
			Handler: getUserListSubscriptionsHandler,
		},
	}

	get_suprsend_user_objects_subscriptions := &Tool{
		Tool: &mcpsdk.Tool{
			Name: "get_suprsend_user_objects_subscriptions",
			Description: `List the objects this user is subscribed TO — what the user follows.

When to use: the user asks "what does X follow?", "what projects is X in?", or you need to enumerate a user's outbound subscriptions.

When NOT to use:
- For followers OF an object (inverse direction) — use get_suprsend_object_subscriptions.
- For mailing-list / segment membership — use get_suprsend_user_list_subscriptions.

Returns: a paginated list of {object_type, object_id} entries. Default limit is 20.`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"distinct_id": {Type: "string", Description: "The distinct_id of the user to get the object subscriptions for."},
					"workspace":   {Type: "string", Description: "SuprSend workspace to run the query from."},
					"limit":       {Type: "number", Description: "Number of object subscriptions to get for a user."},
				},
				Required: []string{"distinct_id"},
			},
			Annotations: mcpsdk.Annotations{
				ReadOnlyHint:   true,
				IdempotentHint: true,
				OpenWorldHint:  mcpsdk.BoolPtr(true),
			},
			Handler: getUserObjectsSubscriptionsHandler,
		},
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
