package tools

import (
	"context"
	"errors"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/cli/pkg/mcpsdk"
	"github.com/suprsend/cli/pkg/mcpserver"
	"github.com/suprsend/suprsend-go"
	"gopkg.in/yaml.v3"
)

func getObjectHandler(ctx context.Context, args mcpsdk.Args) (mcpsdk.Result, error) {
	object_id, err := args.RequireString("object_id")
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	object_type, err := args.RequireString("object_type")
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	workspace := args.GetString("workspace", "staging")
	suprsend_client, err := utils.GetSuprSendWorkspaceClient(workspace, ctx)
	if err != nil {
		if utils.IsAuthError(err) {
			mcpserver.MarkSessionDead(ctx)
		}
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	obj_identifier := suprsend.ObjectIdentifier{
		ObjectType: object_type,
		Id:         object_id,
	}

	objects_resp, err := suprsend_client.Objects.Get(ctx, obj_identifier)
	if err != nil {
		if utils.IsAuthError(err) {
			mcpserver.MarkSessionDead(ctx)
		}
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	yamlobject, err := yaml.Marshal(objects_resp)
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	return mcpsdk.Result{Text: string(yamlobject)}, nil
}

func upsertObjectHandler(ctx context.Context, args mcpsdk.Args) (mcpsdk.Result, error) {
	object_id, err := args.RequireString("object_id")
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	object_type, err := args.RequireString("object_type")
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	workspace := args.GetString("workspace", "staging")
	action, err := args.RequireString("action")
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	suprsend_client, err := utils.GetSuprSendWorkspaceClient(workspace, ctx)
	if err != nil {
		if utils.IsAuthError(err) {
			mcpserver.MarkSessionDead(ctx)
		}
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	obj_identifier := suprsend.ObjectIdentifier{
		ObjectType: object_type,
		Id:         object_id,
	}

	obj_instance := suprsend_client.Objects.GetEditInstance(obj_identifier)

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

	webpush_details, err := getWebpushDetailsFromArgs(args, action)
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	out, err := utils.HandleObjectAction(ctx, obj_instance, action, key, value, slack_details, ms_teams_details, webpush_details, obj_identifier, workspace)
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	_, err = suprsend_client.Objects.Edit(ctx, suprsend.ObjectEditRequest{EditInstance: obj_instance})
	if err != nil {
		if utils.IsAuthError(err) {
			mcpserver.MarkSessionDead(ctx)
		}
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}
	return mcpsdk.Result{Text: out}, nil
}

func getObjectPreferences(ctx context.Context, args mcpsdk.Args) (mcpsdk.Result, error) {
	objId, err := args.RequireString("object_id")
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	objType, err := args.RequireString("object_type")
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	category := args.GetString("category", "")
	channel_preferences := args.GetBool("channel_preferences", false)
	workspace := args.GetString("workspace", "staging")
	suprsendClient, err := utils.GetSuprSendWorkspaceClient(workspace, ctx)
	if err != nil {
		if utils.IsAuthError(err) {
			mcpserver.MarkSessionDead(ctx)
		}
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	objIdentifier := suprsend.ObjectIdentifier{
		Id:         objId,
		ObjectType: objType,
	}

	var objPref interface{}
	if channel_preferences {
		objPref, err = suprsendClient.Objects.GetGlobalChannelsPreference(ctx, objIdentifier, nil)
		if err != nil {
			if utils.IsAuthError(err) {
				mcpserver.MarkSessionDead(ctx)
			}
			return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
		}
	} else if category == "" {
		objPref, err = suprsendClient.Objects.GetFullPreference(ctx, objIdentifier, nil)
		if err != nil {
			if utils.IsAuthError(err) {
				mcpserver.MarkSessionDead(ctx)
			}
			return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
		}
	} else {
		objPref, err = suprsendClient.Objects.GetCategoryPreference(ctx, objIdentifier, category, nil)
		if err != nil {
			if utils.IsAuthError(err) {
				mcpserver.MarkSessionDead(ctx)
			}
			return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
		}
	}

	yamlPref, err := yaml.Marshal(objPref)
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	return mcpsdk.Result{Text: string(yamlPref)}, nil
}

func updateObjectCategoryPreference(ctx context.Context, args mcpsdk.Args) (mcpsdk.Result, error) {
	objId, err := args.RequireString("object_id")
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	objType, err := args.RequireString("object_type")
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	obj := suprsend.ObjectIdentifier{
		Id:         objId,
		ObjectType: objType,
	}

	category, err := args.RequireString("category")
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	rawArgs := args.Map()

	pref, err := args.RequireString("preference")
	if err != nil {
		return mcpsdk.Result{Text: "preference must be a string", IsError: true}, nil
	}

	optOutAny, ok := rawArgs["opt_out_channels"]
	if !ok {
		optOutAny = []any{}
	}
	optOutSlice, ok := optOutAny.([]any)
	if !ok {
		return mcpsdk.Result{Text: "opt_out_channels must be an array", IsError: true}, nil
	}

	optOutChannels := make([]string, 0, len(optOutSlice))
	for _, v := range optOutSlice {
		s, ok := v.(string)
		if !ok {
			return mcpsdk.Result{Text: "opt_out_channels must be an array of strings", IsError: true}, nil
		}
		optOutChannels = append(optOutChannels, s)
	}

	prefPayload := suprsend.ObjectUpdateCategoryPreferenceBody{
		Preference:     pref,
		OptOutChannels: optOutChannels,
	}

	workspace := args.GetString("workspace", "staging")

	suprsendClient, err := utils.GetSuprSendWorkspaceClient(workspace, ctx)
	if err != nil {
		if utils.IsAuthError(err) {
			mcpserver.MarkSessionDead(ctx)
		}
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	objPref, err := suprsendClient.Objects.UpdateCategoryPreference(ctx, obj, category, prefPayload, nil)
	if err != nil {
		if utils.IsAuthError(err) {
			mcpserver.MarkSessionDead(ctx)
		}
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	yamlPref, err := yaml.Marshal(objPref)
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	return mcpsdk.Result{Text: string(yamlPref)}, nil
}

func updateObjectChannelPreferenceHandler(ctx context.Context, args mcpsdk.Args) (mcpsdk.Result, error) {
	objId, err := args.RequireString("object_id")
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	objType, err := args.RequireString("object_type")
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}
	obj := suprsend.ObjectIdentifier{
		Id:         objId,
		ObjectType: objType,
	}

	channelPreferencesAny, ok := args.Map()["channel_preferences"].([]any)
	if !ok {
		return mcpsdk.Result{Text: "channel_preferences must be an array", IsError: true}, nil
	}
	var channelPreferences = []suprsend.ObjectGlobalChannelPreference{}
	err = utils.Remarshal(channelPreferencesAny, &channelPreferences)
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	prefPayload := suprsend.ObjectGlobalChannelsPreferenceUpdateBody{
		ChannelPreferences: channelPreferences,
	}
	workspace := args.GetString("workspace", "staging")
	suprsendClient, err := utils.GetSuprSendWorkspaceClient(workspace, ctx)
	if err != nil {
		if utils.IsAuthError(err) {
			mcpserver.MarkSessionDead(ctx)
		}
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	objPref, err := suprsendClient.Objects.UpdateGlobalChannelsPreference(ctx, obj, prefPayload, nil)
	if err != nil {
		if utils.IsAuthError(err) {
			mcpserver.MarkSessionDead(ctx)
		}
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	yamlPref, err := yaml.Marshal(objPref)
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	return mcpsdk.Result{Text: string(yamlPref)}, nil
}

// getSlackDetailsFromArgs is the mcpsdk.Args sibling of user.go's
// getSlackDetails. Once user.go is ported (Task 5.4) this and the legacy
// helper collapse into a single utility.
func getSlackDetailsFromArgs(args mcpsdk.Args, action string) (map[string]any, error) {
	if action != "add_slack" && action != "remove_slack" {
		return nil, nil
	}
	raw, ok := args.Map()["slack_details"]
	if !ok {
		return nil, errors.New("required argument 'slack_details' not found")
	}
	details, ok := raw.(map[string]any)
	if !ok {
		return nil, errors.New("invalid slack_details")
	}
	return details, nil
}

func getMSTeamsDetailsFromArgs(args mcpsdk.Args, action string) (map[string]any, error) {
	if action != "add_ms_teams" && action != "remove_ms_teams" {
		return nil, nil
	}
	raw, ok := args.Map()["ms_teams_details"]
	if !ok {
		return nil, errors.New("required argument 'ms_teams_details' not found")
	}
	details, ok := raw.(map[string]any)
	if !ok {
		return nil, errors.New("invalid ms_teams_details")
	}
	return details, nil
}

func getWebpushDetailsFromArgs(args mcpsdk.Args, action string) (map[string]any, error) {
	if action != "add_webpush" && action != "remove_webpush" {
		return nil, nil
	}
	raw, ok := args.Map()["webpush_details"]
	if !ok {
		return nil, errors.New("required argument 'webpush_details' not found")
	}
	details, ok := raw.(map[string]any)
	if !ok {
		return nil, errors.New("invalid webpush_details")
	}
	return details, nil
}

// slackDetailsSchema mirrors user.go's slackPropertiesSchema as a typed
// *jsonschema.Schema. Promoted to a package-level var so it can be reused once
// user.go is ported in Task 5.4.
var slackDetailsSchema = &jsonschema.Schema{
	Type:        "object",
	Description: "This is only applicable for add_slack and remove_slack actions.",
	Properties: map[string]*jsonschema.Schema{
		"access_token":               {Type: "string", Description: "Access token for the Slack workspace"},
		"slack_email":                {Type: "string", Description: "Email of the user to add to the Slack workspace"},
		"slack_channel_id":           {Type: "string", Description: "ID of the Slack channel"},
		"slack_user_id":              {Type: "string", Description: "ID of the Slack user"},
		"slack_incoming_webhook_url": {Type: "string", Description: "Incoming webhook URL for the Slack"},
	},
}

// msTeamsDetailsSchema mirrors user.go's msTeamsPropertiesSchema (+ the
// msTeamsRequiredFields() option that stamps `required: ["type"]`) as a
// single typed *jsonschema.Schema.
var msTeamsDetailsSchema = &jsonschema.Schema{
	Type:        "object",
	Description: "This is only applicable for add_ms_teams and remove_ms_teams actions.",
	Properties: map[string]*jsonschema.Schema{
		"type": {
			Type: "string",
			Enum: []any{"incoming_webhook", "channel", "user", "user_id"},
		},
		"incoming_webhook": {
			Type:                 "object",
			Title:                "IncomingWebhook",
			AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}},
			Properties: map[string]*jsonschema.Schema{
				"url": {Type: "string", Format: "uri"},
			},
			Required: []string{"url"},
		},
		"channel": {
			Type:                 "object",
			Title:                "Channel",
			AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}},
			Properties: map[string]*jsonschema.Schema{
				"tenant_id":       {Type: "string"},
				"service_url":     {Type: "string", Format: "uri"},
				"conversation_id": {Type: "string"},
			},
			Required: []string{"conversation_id", "service_url", "tenant_id"},
		},
		"user": {
			Type:                 "object",
			Title:                "Channel",
			AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}},
			Properties: map[string]*jsonschema.Schema{
				"tenant_id":       {Type: "string"},
				"service_url":     {Type: "string", Format: "uri"},
				"conversation_id": {Type: "string"},
			},
			Required: []string{"conversation_id", "service_url", "tenant_id"},
		},
		"user_id": {
			Type:                 "object",
			Title:                "UserID",
			AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}},
			Properties: map[string]*jsonschema.Schema{
				"tenant_id":   {Type: "string"},
				"service_url": {Type: "string", Format: "uri"},
				"user_id":     {Type: "string"},
			},
			Required: []string{"tenant_id", "user_id", "service_url"},
		},
	},
	Required: []string{"type"},
}

// webpushDetailsSchema rewrites the inline mcp.WithObject(...) literal as a
// typed *jsonschema.Schema. Keys.auth + keys.p256dh are both required and the
// `keys` object disallows additional properties (matching the original
// `additionalProperties: false`).
var webpushDetailsSchema = &jsonschema.Schema{
	Type:        "object",
	Description: "This is only applicable for add_webpush and remove_webpush actions.",
	Properties: map[string]*jsonschema.Schema{
		"keys": {
			Type:                 "object",
			AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}},
			Properties: map[string]*jsonschema.Schema{
				"auth":   {Type: "string", Description: "The auth key for the webpush"},
				"p256dh": {Type: "string", Description: "The p256dh key for the webpush"},
			},
			Required: []string{"auth", "p256dh"},
		},
		"endpoint": {Type: "string", Description: "The endpoint for the webpush"},
	},
}

func newObjectTools() []*Tool {
	get_suprsend_object := &Tool{
		Tool: &mcpsdk.Tool{
			Name: "get_suprsend_object",
			Description: `Get a SuprSend object's full state by object_type + object_id. Objects are non-user entities — organizations, projects, vehicles, devices — namespaced by object_type.

When to use: the user references an object by id and you need its stored properties or channel identifiers.

When NOT to use:
- For users — use get_suprsend_user.
- For preferences only — use get_suprsend_object_preferences.
- For followers / members — use get_suprsend_object_subscriptions.

Returns: YAML mirroring get_suprsend_user's shape — object_type, object_id, properties (custom fields), created_at, updated_at, and a channels array (each entry has channel value, status, perma_status).`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"object_id":   {Type: "string", Description: "The object_id of the object to get."},
					"object_type": {Type: "string", Description: "The type of object you want to get."},
					"workspace":   {Type: "string", Description: "Suprsend workspace to get the object from"},
				},
				Required: []string{"object_id", "object_type"},
			},
			Annotations: mcpsdk.Annotations{
				ReadOnlyHint:   true,
				IdempotentHint: true,
				OpenWorldHint:  true,
			},
			Handler: getObjectHandler,
		},
	}

	upsert_suprsend_object := &Tool{
		Tool: &mcpsdk.Tool{
			Name: "upsert_suprsend_object",
			Description: `Modify properties or channel identifiers on a SuprSend object — a non-user entity like an organization, project, or vehicle. One call performs ONE action; for multiple changes, call this tool multiple times.

Actions:
- set, set_once, unset, remove — modify a scalar property by key/value. remove permanently deletes the key.
- append, increment — modify array / numeric values.
- add_email / remove_email, add_sms / remove_sms, add_whatsapp / remove_whatsapp, add_androidpush / remove_androidpush, add_iospush / remove_iospush, add_slack / remove_slack, add_ms_teams / remove_ms_teams, add_webpush / remove_webpush — register or deregister a delivery channel.

Channel registration is special. For channel identifiers ALWAYS use the dedicated add_<channel> / remove_<channel> actions — generic set / unset will not register the channel correctly with the delivery router. Slack and MS Teams additionally require the corresponding slack_details / ms_teams_details payload alongside the action; channel id alone is not enough.

When to use: creating an object or modifying its stored state — properties, channels, identifiers.

When NOT to use:
- For users — use upsert_suprsend_user instead.
- For preferences — use update_suprsend_category_preference_object (per category) or update_suprsend_object_channel_preference (across categories).
- For followers / members — use add_suprsend_object_subscriptions.

Side effects: remove and unset permanently delete data. add_<channel> makes the object reachable on that channel for any future workflow run; remove_<channel> stops delivery immediately.

object_type namespaces the object (e.g., "organization", "project") and is required.

Returns: the updated object on success; structured error with field reasons on failure.`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"object_id":   {Type: "string", Description: "The object_id of the object to get."},
					"object_type": {Type: "string", Description: "The type of object you want to get."},
					"workspace":   {Type: "string", Description: "Suprsend workspace to get the object from."},
					"object_payload": {
						Type:        "object",
						Description: "Payload of the request that you want to pass for the object.",
					},
					"action": {
						Type: "string",
						Description: `
					the action to perform.
					use action "upsert" to create a new object or update an existing object's properties.
					use action "remove" to remove a object's properties.
					use action "set" to set a object's property, don't use this when trying to add email, add sms, add whatsapp, add androidpush, add iospush, add slack, add ms_teams, add webpush use the respective actions.
					use action "unset" to unset a object's property, don't use this when trying to remove email, remove sms, remove whatsapp, remove androidpush, remove iospush, remove slack, remove ms_teams, remove webpush use the respective actions.
					use action "set_once" to set a object's property once, this will only set the property if it is not already set.
					use action "append" to append a value to a object's property.
					use action "increment" to increment a object's property.
					use action "add_email" to add an email to a object.
					use action "remove_email" to remove an email from a object.
					use action "add_sms" to add an SMS to a object.
					use action "remove_sms" to remove an SMS from a object.
					use action "add_whatsapp" to add a WhatsApp to a object.
					use action "remove_whatsapp" to remove a WhatsApp from a object.
					use action "add_androidpush" to add an Android push to a object.
					use action "remove_androidpush" to remove an Android push from a object.
					use action "add_iospush" to add an iOS push to a object.
					use action "remove_iospush" to remove an iOS push from a object.
					use action "add_slack" to add a Slack to a object.
					use action "remove_slack" to remove a Slack from a object.
					use action "add_ms_teams" to add a Microsoft Teams to a object.
					use action "remove_ms_teams" to remove a Microsoft Teams from a object.
					use action "add_webpush" to add a Webpush to a object.
					use action "remove_webpush" to remove a Webpush from a object.
					`,
						Enum: []any{
							"upsert",
							"remove",
							"set",
							"unset",
							"set_once",
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
				Required: []string{"object_id", "object_type", "action"},
			},
			Annotations: mcpsdk.Annotations{
				DestructiveHint: true,
				IdempotentHint:  false,
				OpenWorldHint:   true,
			},
			Handler: upsertObjectHandler,
		},
	}

	get_suprsend_object_preferences := &Tool{
		Tool: &mcpsdk.Tool{
			Name: "get_suprsend_object_preferences",
			Description: `Read an object's category-level notification preferences and (optionally) per-channel overrides.

When to use:
- Before update_suprsend_category_preference_object, to read current state.
- Before sending to an object, to check delivery permission.

When NOT to use:
- For the object's identity or channels — use get_suprsend_object.
- For users — use get_suprsend_user_preferences.

Returns: the object's preference tree. Pass category to scope to one preference; omit for the full tree. Set channel_preferences=true to include per-channel overrides.`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"object_id":           {Type: "string", Description: "The object_id of the object to get preferences from."},
					"object_type":         {Type: "string", Description: "The object_type of the object to get preferences from."},
					"category":            {Type: "string", Description: "The category_slug of the object to get preferences from, if not provided, it will get all the preferences for the object."},
					"channel_preferences": {Type: "boolean", Description: "set this to true to get all the channel preferences for the object."},
					"workspace":           {Type: "string", Description: "SuprSend workspace to get the user from."},
				},
				Required: []string{"object_id", "object_type"},
			},
			Annotations: mcpsdk.Annotations{
				ReadOnlyHint:   true,
				IdempotentHint: true,
				OpenWorldHint:  true,
			},
			Handler: getObjectPreferences,
		},
	}

	update_suprsend_category_preference_object := &Tool{
		Tool: &mcpsdk.Tool{
			Name: "update_suprsend_category_preference_object",
			Description: `Set ONE category's preference for ONE object — opted in, opted out, or cant_unsubscribe (locked) — plus per-channel opt-outs within that category.

Replaces, does not merge. This call overwrites the existing preference for the named category. Previous opt-outs within the same category are lost; pass them again in opt_out_channels if you want to keep them.

When to use: changing a single category on a single object.

When NOT to use:
- For object-wide channel toggles ("block all SMS") — use update_suprsend_object_channel_preference.
- For users — use update_suprsend_users_preferences.
- For tenant defaults — use update_suprsend_tenant_default_preference.

Preference values: opt_in enables; opt_out disables; cant_unsubscribe locks the object from toggling this category.

Returns: updated preference state on success; structured error on failure.`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"object_id":   {Type: "string", Description: "The object_id of the object to get preferences from."},
					"object_type": {Type: "string", Description: "The object_type of the object to get preferences from."},
					"category":    {Type: "string", Description: "category_slug of an category to get."},
					"preference": {
						Type:        "string",
						Description: "The preference to update for the object.",
						Enum:        []any{"opt_in", "opt_out"},
					},
					"opt_out_channels": {
						Type:        "array",
						Description: "The channels to opt out from for the object.",
						Items:       &jsonschema.Schema{Type: "string"},
					},
					"workspace": {Type: "string", Description: "SuprSend workspace to get the user from."},
				},
				Required: []string{"object_id", "object_type", "category", "preference"},
			},
			Annotations: mcpsdk.Annotations{
				DestructiveHint: true,
				IdempotentHint:  true,
				OpenWorldHint:   true,
			},
			Handler: updateObjectCategoryPreference,
		},
	}

	update_suprsend_object_channel_preference := &Tool{
		Tool: &mcpsdk.Tool{
			Name: "update_suprsend_object_channel_preference",
			Description: `Block or allow specific delivery channels for ONE object, applied across ALL categories.

is_restricted semantics: true blocks delivery on that channel; false re-enables it. Each entry in channel_preferences is a {channel, is_restricted} pair.

When NOT to use:
- For per-category control — use update_suprsend_category_preference_object.
- For users — use update_suprsend_user_channel_preference.

Side effects: takes effect on the next workflow run; in-flight notifications may still send.

Returns: updated channel-preference state on success.`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"object_id":   {Type: "string", Description: "The object_id of the object to update the channel preference for."},
					"object_type": {Type: "string", Description: "The object_type of the object to update the channel preference for."},
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
					"workspace": {Type: "string", Description: "SuprSend workspace to update the channel preference for."},
				},
				Required: []string{"object_id", "object_type", "channel_preferences"},
			},
			Annotations: mcpsdk.Annotations{
				DestructiveHint: true,
				IdempotentHint:  true,
				OpenWorldHint:   true,
			},
			Handler: updateObjectChannelPreferenceHandler,
		},
	}

	// NOTE: get_suprsend_object_subscriptions + add_suprsend_object_subscriptions
	// (handlers + tool literals) moved to obj_subscriptions.go as part of the
	// Phase-5 mcpsdk port (Task 5.2). They register themselves via their own
	// init() under type "objects", so the surface stays unchanged.

	tools := []*Tool{
		get_suprsend_object,
		upsert_suprsend_object,
		get_suprsend_object_preferences,
		update_suprsend_category_preference_object,
		update_suprsend_object_channel_preference,
	}

	return tools
}

func init() {
	for _, t := range newObjectTools() {
		RegisterTool(t, "objects")
	}
}
