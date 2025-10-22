package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/suprsend-go"
	"gopkg.in/yaml.v3"
)

func getObjectHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	object_id, err := request.RequireString("object_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	object_type, err := request.RequireString("object_type")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	workspace := request.GetString("workspace", "staging")
	suprsend_client, err := utils.GetSuprSendWorkspaceClient(workspace)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	obj_identifier := suprsend.ObjectIdentifier{
		ObjectType: object_type,
		Id:         object_id,
	}

	objects_resp, err := suprsend_client.Objects.Get(ctx, obj_identifier)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	yamlobject, err := yaml.Marshal(objects_resp)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(string(yamlobject)), nil
}

func upsertObjectHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	object_id, err := request.RequireString("object_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	object_type, err := request.RequireString("object_type")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	workspace := request.GetString("workspace", "staging")
	action, err := request.RequireString("action")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	suprsend_client, err := utils.GetSuprSendWorkspaceClient(workspace)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	obj_identifier := suprsend.ObjectIdentifier{
		ObjectType: object_type,
		Id:         object_id,
	}

	obj_instance := suprsend_client.Objects.GetEditInstance(obj_identifier)

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

	identity_provider := request.GetString("identity_provider", "")

	out, err := utils.HandleObjectAction(ctx, obj_instance, action, key, value, slack_details, identity_provider, obj_identifier, workspace)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(out), nil
}

func getObjectPreferences(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objId, err := request.RequireString("object_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	objType, err := request.RequireString("object_type")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	category := request.GetString("category", "")
	channel_preferences := request.GetBool("channel_preferences", false)
	workspace := request.GetString("workspace", "staging")
	suprsendClient, err := utils.GetSuprSendWorkspaceClient(workspace)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	objIdentifier := suprsend.ObjectIdentifier{
		Id:         objId,
		ObjectType: objType,
	}

	var objPref interface{}
	if category == "" {
		objPref, err = suprsendClient.Objects.GetFullPreference(ctx, objIdentifier, nil)
		if err != nil {
			return nil, err
		}
	} else {
		objPref, err = suprsendClient.Objects.GetCategoryPreference(ctx, objIdentifier, category, nil)
		if err != nil {
			return nil, err
		}
	}
	if channel_preferences {
		objPref, err = suprsendClient.Objects.GetGlobalChannelsPreference(ctx, objIdentifier, nil)
		if err != nil {
			return nil, err
		}
	}

	yamlPref, err := yaml.Marshal(objPref)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(yamlPref)), nil
}

func updateObjectCategoryPreference(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objId, err := request.RequireString("object_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	objType, err := request.RequireString("object_type")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	obj := suprsend.ObjectIdentifier{
		Id:         objId,
		ObjectType: objType,
	}

	category, err := request.RequireString("category")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args := request.GetArguments()

	pref, err := request.RequireString("preference")
	if err != nil {
		return mcp.NewToolResultError("preference must be a string"), nil
	}

	optOutAny, ok := args["opt_out_channels"]
	if !ok {
		optOutAny = []any{}
	}
	optOutSlice, ok := optOutAny.([]any)
	if !ok {
		return mcp.NewToolResultError("opt_out_channels must be an array"), nil
	}

	optOutChannels := make([]string, 0, len(optOutSlice))
	for _, v := range optOutSlice {
		s, ok := v.(string)
		if !ok {
			return mcp.NewToolResultError("opt_out_channels must be an array of strings"), nil
		}
		optOutChannels = append(optOutChannels, s)
	}

	prefPayload := suprsend.ObjectUpdateCategoryPreferenceBody{
		Preference:     pref,
		OptOutChannels: optOutChannels,
	}

	workspace := request.GetString("workspace", "staging")

	suprsendClient, err := utils.GetSuprSendWorkspaceClient(workspace)
	if err != nil {
		return nil, err
	}

	objPref, err := suprsendClient.Objects.UpdateCategoryPreference(ctx, obj, category, prefPayload, nil)
	if err != nil {
		return nil, err
	}

	yamlPref, err := yaml.Marshal(objPref)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(yamlPref)), nil
}

func updateObjectChannelPreferenceHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objId, err := request.RequireString("object_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	objType, err := request.RequireString("object_type")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	obj := suprsend.ObjectIdentifier{
		Id:         objId,
		ObjectType: objType,
	}

	rawPayload, ok := request.GetArguments()["payload"].(map[string]any)
	if !ok {
		return mcp.NewToolResultError("payload must be an object"), nil
	}
	channelPreferences, ok := rawPayload["channel_preferences"].([]suprsend.ObjectGlobalChannelPreference)
	if !ok {
		return mcp.NewToolResultError("channel_preferences must be an array"), nil
	}

	prefPayload := suprsend.ObjectGlobalChannelsPreferenceUpdateBody{
		ChannelPreferences: channelPreferences,
	}
	workspace := request.GetString("workspace", "staging")
	suprsendClient, err := utils.GetSuprSendWorkspaceClient(workspace)
	if err != nil {
		return nil, err
	}

	objPref, err := suprsendClient.Objects.UpdateGlobalChannelsPreference(ctx, obj, prefPayload, nil)
	if err != nil {
		return nil, err
	}

	yamlPref, err := yaml.Marshal(objPref)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(yamlPref)), nil
}

func newObjectTools() []*Tool {
	get_suprsend_object := &Tool{
		Name:        "objects.get",
		Description: "Enables querying object information",
		MCPTool: mcp.NewTool("get_suprsend_object",
			mcp.WithDescription("Use this tool to get all details about an object"),
			mcp.WithString("object_id",
				mcp.Description("The object_id of the object to get."),
				mcp.Required(),
			),
			mcp.WithString("object_type",
				mcp.Description("The type of object you want to get."),
				mcp.Required(),
			),
			mcp.WithString("workspace",
				mcp.Description("Suprsend workspace to get the object from"),
			),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		Handler: getObjectHandler,
	}

	upsert_suprsend_object := &Tool{
		Name:        "objects.upsert",
		Description: "Enables upserting object information",
		MCPTool: mcp.NewTool("upsert_suprsend_object",
			mcp.WithDescription("Use this tool to upsert an object"),
			mcp.WithString("object_id",
				mcp.Description("The object_id of the object to get."),
				mcp.Required(),
			),
			mcp.WithString("object_type",
				mcp.Description("The type of object you want to get."),
				mcp.Required(),
			),
			mcp.WithString("workspace",
				mcp.Description("Suprsend workspace to get the object from."),
			),
			mcp.WithObject("object_payload",
				mcp.Description("Payload of the request that you want to pass for the object."),
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
		Handler: upsertObjectHandler,
	}

	get_suprsend_object_preferences := &Tool{
		Name:        "objects.get_preferences",
		Description: "Enables querying object preferences(also within a specific category)",
		MCPTool: mcp.NewTool("get_suprsend_object_preferences",
			mcp.WithDescription("Use this tool to get the preferences(also within a specific category) for an object."),
			mcp.WithString("object_id",
				mcp.Description("The object_id of the object to get preferences from."),
				mcp.Required(),
			),
			mcp.WithString("object_type",
				mcp.Description("The object_type of the object to get preferences from."),
				mcp.Required(),
			),
			mcp.WithString("category",
				mcp.Description("The category_slug of the object to get preferences from"),
			),
			mcp.WithBoolean("channel_preferences",
				mcp.Description("Whether to include channel preferences in the response. Default is false."),
			),
			mcp.WithString("workspace",
				mcp.Description("SuprSend workspace to get the user from."),
			),
		),
		Handler: getObjectPreferences,
	}

	update_suprsend_category_preference_object := &Tool{
		Name:        "objects.update_preferences",
		Description: "Enables updating a specific category preference for an object, controlling notification preferences and channel opt-outs.",
		MCPTool: mcp.NewTool("update_suprsend_category_preference_object",
			mcp.WithDescription("Use this tool to update a specific category preference for an object, controlling notification preferences and channel opt-outs."),
			mcp.WithString("object_id",
				mcp.Description("The object_id of the object to get preferences from."),
				mcp.Required(),
			),
			mcp.WithString("object_type",
				mcp.Description("The object_type of the object to get preferences from."),
				mcp.Required(),
			),
			mcp.WithString("category",
				mcp.Description("category_slug of an category to get."),
				mcp.Required(),
			),
			mcp.WithString("preference",
				mcp.Enum(
					"opt_in",
					"opt_out",
				),
				mcp.Description("The preference to update for the object."),
				mcp.Required(),
			),
			mcp.WithArray("opt_out_channels",
				mcp.Description("The channels to opt out from for the object."),
				mcp.WithStringItems(),
			),
			mcp.WithString("workspace",
				mcp.Description("SuprSend workspace to get the user from."),
			),
			mcp.WithDestructiveHintAnnotation(true),
		),
		Handler: updateObjectCategoryPreference,
	}

	update_suprsend_object_channel_preference := &Tool{
		Name:        "objects.update_channel_preference",
		Description: "Enables updating channel preference for an object",
		MCPTool: mcp.NewTool("update_suprsend_object_channel_preference",
			mcp.WithDescription("Use this tool to update channel preference for an object."),
			mcp.WithString("object_id",
				mcp.Description("The object_id of the object to update the channel preference for."),
				mcp.Required(),
			),
			mcp.WithString("object_type",
				mcp.Description("The object_type of the object to update the channel preference for."),
				mcp.Required(),
			),
			mcp.WithObject("payload",
				mcp.Description("The channel preference to update for the object."),
				mcp.Required(),
			),
			mcp.WithString("workspace",
				mcp.Description("SuprSend workspace to update the channel preference for."),
			),
			mcp.WithDestructiveHintAnnotation(true),
		),
		Handler: updateObjectChannelPreferenceHandler,
	}

	get_suprsend_obj_subscriptions := &Tool{
		Name:        "object.get_subscriptions",
		Description: "Enables querying subscriptions of an object",
		MCPTool: mcp.NewTool("get_suprsend_object_subscriptions",
			mcp.WithDescription("Use this tool to get all details about the subscriptions of an object"),
			mcp.WithString("object_id",
				mcp.Description("The object_id of the object's subscriptions to get."),
				mcp.Required(),
			),
			mcp.WithString("object_type",
				mcp.Description("The type of object you want to get."),
				mcp.Required(),
			),
			mcp.WithString("workspace",
				mcp.Description("Suprsend workspace to get the object from."),
			),
			mcp.WithBoolean("channel_preferences",
				mcp.Description("Whether to include channel preferences in the response. Default is false."),
			),
			mcp.WithNumber("limit",
				mcp.Description("Number of subscriptions to get for an object."),
			),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		Handler: getObjectSubscriptionsHandler,
	}

	add_suprsend_obj_subscriptions := &Tool{
		Name:        "object.upsert_subscriptions",
		Description: "Enables upserting subscription to an object. Allows users or other objects to subscribe to an object.",
		MCPTool: mcp.NewTool("add_suprsend_object_subscriptions",
			mcp.WithDescription("Use this tool to add subscriptions to an object. Allows users or other objects to subscribe to an object."),
			mcp.WithString("object_id",
				mcp.Description("The object_id of the object's subscriptions to get."),
				mcp.Required(),
			),
			mcp.WithString("object_type",
				mcp.Description("The type of object you want to get."),
				mcp.Required(),
			),
			mcp.WithString("workspace",
				mcp.Description("Suprsend workspace to get the object from."),
			),
			mcp.WithArray("recipients",
				mcp.Description("Users & Objects who are subscribing to an object"),
				mcp.Required(),
			),
			mcp.WithObject("properties",
				mcp.Description("Properties of an user/object"),
			),
			mcp.WithDestructiveHintAnnotation(true),
		),
		Handler: addObjectSubscriptionsHandler,
	}
	tools := []*Tool{
		get_suprsend_object,
		upsert_suprsend_object,
		get_suprsend_obj_subscriptions,
		add_suprsend_obj_subscriptions,
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
