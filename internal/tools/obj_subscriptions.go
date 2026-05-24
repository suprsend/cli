package tools

import (
	"context"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/cli/pkg/mcpsdk"
	"github.com/suprsend/suprsend-go"
	"gopkg.in/yaml.v3"
)

func getObjectSubscriptionsHandler(ctx context.Context, args mcpsdk.Args) (mcpsdk.Result, error) {
	object_id, err := args.RequireString("object_id")
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	object_type, err := args.RequireString("object_type")
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	limit_subscriptions := args.GetInt("limit", 20)
	cursor_list_api_opts := suprsend.CursorListApiOptions{
		Limit: limit_subscriptions,
	}
	workspace := args.GetString("workspace", "staging")

	suprsend_client, err := utils.GetSuprSendWorkspaceClient(workspace, ctx)
	if err != nil {
		if utils.IsAuthError(err) {
			markSessionDead(ctx)
		}
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	obj_identifier := suprsend.ObjectIdentifier{
		ObjectType: object_type,
		Id:         object_id,
	}

	obj_subs_resp, err := suprsend_client.Objects.GetSubscriptions(ctx, obj_identifier, &cursor_list_api_opts)
	if err != nil {
		if utils.IsAuthError(err) {
			markSessionDead(ctx)
		}
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	yamlobject, err := yaml.Marshal(obj_subs_resp)
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	return mcpsdk.Result{Text: string(yamlobject)}, nil
}

func addObjectSubscriptionsHandler(ctx context.Context, args mcpsdk.Args) (mcpsdk.Result, error) {
	object_id, err := args.RequireString("object_id")
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	object_type, err := args.RequireString("object_type")
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	workspace := args.GetString("workspace", "staging")

	recipients, ok := args.Map()["recipients"]
	if !ok {
		return mcpsdk.Result{Text: "Recipients is a required property", IsError: true}, nil
	}

	props, ok := args.Map()["properties"]
	if !ok {
		return mcpsdk.Result{Text: "Properties isn't passed as an object", IsError: true}, nil
	}

	suprsend_client, err := utils.GetSuprSendWorkspaceClient(workspace, ctx)
	if err != nil {
		if utils.IsAuthError(err) {
			markSessionDead(ctx)
		}
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	obj_identifier := suprsend.ObjectIdentifier{
		ObjectType: object_type,
		Id:         object_id,
	}

	payload := map[string]any{
		"recipients": recipients,
		"properties": props,
	}

	obj_subs_resp, err := suprsend_client.Objects.CreateSubscriptions(ctx, obj_identifier, payload)
	if err != nil {
		if utils.IsAuthError(err) {
			markSessionDead(ctx)
		}
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	yamlobject, err := yaml.Marshal(obj_subs_resp)
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	return mcpsdk.Result{Text: string(yamlobject)}, nil
}

func newObjSubscriptionsTools() []*Tool {
	get_suprsend_obj_subscriptions := &Tool{
		Tool: &mcpsdk.Tool{
			Name: "get_suprsend_object_subscriptions",
			Description: `List users / objects subscribed TO this object (its followers / members). Subscriptions are stored on the followed object.

When to use: the user asks "who follows project X?", "who's a member of organization Y?", or you need to enumerate an object's inbound subscribers.

When NOT to use:
- For the inverse direction (what a user follows) — use get_suprsend_user_objects_subscriptions.
- For mailing-list members — use get_suprsend_user_list_subscriptions on each user.

Returns: a paginated list of subscriber {type, id} entries. Set channel_preferences=true to also include each subscriber's channel preferences for this object. Default limit is 20.`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"object_id":           {Type: "string", Description: "The object_id of the object's subscriptions to get."},
					"object_type":         {Type: "string", Description: "The type of object you want to get."},
					"workspace":           {Type: "string", Description: "Suprsend workspace to get the object from."},
					"channel_preferences": {Type: "boolean", Description: "Whether to include channel preferences in the response. Default is false."},
					"limit":               {Type: "number", Description: "Number of subscriptions to get for an object."},
				},
				Required: []string{"object_id", "object_type"},
			},
			Annotations: mcpsdk.Annotations{
				ReadOnlyHint:   true,
				IdempotentHint: true,
				OpenWorldHint:  true,
			},
			Handler: getObjectSubscriptionsHandler,
		},
	}

	add_suprsend_obj_subscriptions := &Tool{
		Tool: &mcpsdk.Tool{
			Name: "add_suprsend_object_subscriptions",
			Description: `Subscribe one or more users or other objects TO this object. The recipient list can mix users and objects in a single call.

Recipients: users by distinct_id, objects by object_type + id. Each entry's shape follows the SuprSend recipient format. Optional properties attach metadata to each subscription (role, joined_at, etc.).

When NOT to use:
- To remove subscribers — there is no remove tool; use the SuprSend API directly.
- For mailing-list / segment membership — those are managed via the Lists API.
- For preference changes on existing subscribers — use the per-user / per-object preference tools.

Side effects: each successful subscription is a separate row. Calling this twice with the same recipient creates duplicate-looking entries; check existing state with get_suprsend_object_subscriptions first if duplicates would be a problem.

Returns: the created subscription records on success.`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"object_id":   {Type: "string", Description: "The object_id of the object's subscriptions to get."},
					"object_type": {Type: "string", Description: "The type of object you want to get."},
					"workspace":   {Type: "string", Description: "Suprsend workspace to get the object from."},
					"recipients": {
						Type:        "array",
						Description: "Users & Objects who are subscribing to an object",
					},
					"properties": {
						Type:        "object",
						Description: "Properties of an user/object",
					},
				},
				Required: []string{"object_id", "object_type", "recipients"},
			},
			Annotations: mcpsdk.Annotations{
				DestructiveHint: true,
				IdempotentHint:  false,
				OpenWorldHint:   true,
			},
			Handler: addObjectSubscriptionsHandler,
		},
	}

	return []*Tool{
		get_suprsend_obj_subscriptions,
		add_suprsend_obj_subscriptions,
	}
}

func init() {
	for _, t := range newObjSubscriptionsTools() {
		RegisterTool(t, "objects")
	}
}
