package tools

import (
	"context"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/suprsend/cli/internal/utils"
	suprsend "github.com/suprsend/suprsend-go"
	"gopkg.in/yaml.v3"
)

func getTenantHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tenant_id, err := request.RequireString("tenant_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	workspace := request.GetString("workspace", "staging")

	suprsend_client, err := utils.GetSuprSendWorkspaceClient(workspace)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	// todo: rename these
	user, err := suprsend_client.Tenants.Get(ctx, tenant_id)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	yamluser, err := yaml.Marshal(user)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(string(yamluser)), nil
}

func upsertTenantHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tenant_id, err := request.RequireString("tenant_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	workspace := request.GetString("workspace", "staging")

	suprsend_client, err := utils.GetSuprSendWorkspaceClient(workspace)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	raw_props, ok := request.GetArguments()["tenant_properties"]
	if !ok {
		return mcp.NewToolResultError("tenant_properties is required"), nil
	}

	tenant_properties, ok := raw_props.(map[string]any)
	if !ok {
		return mcp.NewToolResultError("invalid tenant_properties"), nil
	}

	// todo: add blocked_channels
	// blocked_channels, blocked_channels_ok := tenant_properties["blocked_channels"]

	tenant_payload := &suprsend.Tenant{
		TenantName:             utils.GetStringPtr(tenant_properties, "tenant_name"),
		Logo:                   utils.GetStringPtr(tenant_properties, "logo"),
		Timezone:               utils.GetStringPtr(tenant_properties, "timezone"),
		PrimaryColor:           utils.GetStringPtr(tenant_properties, "primary_color"),
		SecondaryColor:         utils.GetStringPtr(tenant_properties, "secondary_color"),
		TertiaryColor:          utils.GetStringPtr(tenant_properties, "tertiary_color"),
		EmbeddedPreferenceUrl:  utils.GetStringPtr(tenant_properties, "embedded_preference_url"),
		HostedPreferenceDomain: utils.GetStringPtr(tenant_properties, "hosted_preference_domain"),
		SocialLinks:            buildSocialLinks(utils.GetMap(tenant_properties, "social_links")),
	}

	if custom_tenant_properties, ok := tenant_properties["custom_properties"].(map[string]any); ok {
		tenant_payload.Properties = custom_tenant_properties
	}

	tenant, err := suprsend_client.Tenants.Upsert(ctx, tenant_id, tenant_payload)
	if err != nil {
		err_str := err.Error()
		if strings.Contains(err_str, `{"tenant_name": "missing value"}`) {
			return mcp.NewToolResultError("tenant_name is required when creating a new tenant. Try again with a tenant_name."), nil
		}
		return mcp.NewToolResultError(err_str), nil
	}

	yamluser, err := yaml.Marshal(tenant)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(string(yamluser)), nil
}

func updateCategoryPreferenceTenant(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tenantId, err := request.RequireString("tenant_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if tenantId == "" {
		return mcp.NewToolResultError("tenant_id is required"), nil
	}

	category, err := request.RequireString("category")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if category == "" {
		return mcp.NewToolResultError("category is required"), nil
	}

	args := request.GetArguments()

	pref, ok := args["preference"].(string)
	if !ok {
		return mcp.NewToolResultError("preference must be a string"), nil
	}

	visibleToSubscriber, ok := args["visible_to_subscriber"].(bool)
	if !ok {
		return mcp.NewToolResultError("visible_to_subscriber must be bool"), nil
	}

	mandatoryChannelsAny, ok := args["mandatory_channels"]
	if !ok {
		mandatoryChannelsAny = []any{}
	}
	mandatoryChannelsSlice, ok := mandatoryChannelsAny.([]any)
	if !ok {
		return mcp.NewToolResultError("mandatory_channels must be an array"), nil
	}

	mandatoryChannels := make([]string, 0, len(mandatoryChannelsSlice))
	for _, v := range mandatoryChannelsSlice {
		s, ok := v.(string)
		if !ok {
			return mcp.NewToolResultError("mandatory_channels must be an array of strings"), nil
		}
		mandatoryChannels = append(mandatoryChannels, s)
	}

	blockedChannelsAny, ok := args["blocked_channels"]
	if !ok {
		blockedChannelsAny = []any{}
	}
	blockedChannelsSlice, ok := blockedChannelsAny.([]any)
	if !ok {
		return mcp.NewToolResultError("blocked_channels must be an array"), nil
	}

	blockedChannels := make([]string, 0, len(blockedChannelsSlice))
	for _, v := range blockedChannelsSlice {
		s, ok := v.(string)
		if !ok {
			return mcp.NewToolResultError("blocked_channels must be an array of strings"), nil
		}
		blockedChannels = append(blockedChannels, s)
	}

	prefPayload := suprsend.TenantCategoryPreferenceUpdateBody{
		Preference:          pref,
		VisibleToSubscriber: &visibleToSubscriber,
		MandatoryChannels:   mandatoryChannels,
		BlockedChannels:     blockedChannels,
	}

	workspace := request.GetString("workspace", "staging")

	suprsendClient, err := utils.GetSuprSendWorkspaceClient(workspace)
	if err != nil {
		return nil, err
	}

	tenantPref, err := suprsendClient.Tenants.UpdateCategoryPreference(ctx, tenantId, category, prefPayload)
	if err != nil {
		return nil, err
	}

	yamlPref, err := yaml.Marshal(tenantPref)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(yamlPref)), nil
}

func getDefaultPreferenceTenant(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tenantId, err := request.RequireString("tenant_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	workspace := request.GetString("workspace", "staging")

	suprsendClient, err := utils.GetSuprSendWorkspaceClient(workspace)
	if err != nil {
		return nil, err
	}

	tenantPref, err := suprsendClient.Tenants.GetAllCategoriesPreference(ctx, tenantId, nil)
	if err != nil {
		return nil, err
	}

	yamlPref, err := yaml.Marshal(tenantPref)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(yamlPref)), nil
}

func getAllTenantsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	workspace := request.GetString("workspace", "staging")
	limit := request.GetInt("limit", 100)

	suprsendClient, err := utils.GetSuprSendWorkspaceClient(workspace)
	if err != nil {
		return nil, err
	}

	tenants, err := suprsendClient.Tenants.List(ctx, &suprsend.TenantListOptions{
		Limit: limit,
	})
	if err != nil {
		return nil, err
	}

	yamlTenants, err := yaml.Marshal(tenants)
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(string(yamlTenants)), nil
}

func newTenantTools() []*Tool {
	get_suprsend_tenant := &Tool{
		Name:        "tenants.get",
		Description: "Enables querying tenant information",
		MCPTool: mcp.NewTool("get_suprsend_tenant",
			mcp.WithDescription(`Use this tool to get all properties for a tenant in SuprSend. If the workspace is not specified. ask the user to provide it before using this tool.`),
			mcp.WithString("tenant_id",
				mcp.Description(`The tenant_id of the tenant to get.`),
				mcp.Required(),
			),
			mcp.WithString("workspace",
				mcp.Description(`SuprSend workspace to get the tenant from.`),
			),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		Handler: getTenantHandler,
	}

	get_suprsend_tenants := &Tool{
		Name:        "tenants.get_all",
		Description: "Enables querying all tenants",
		MCPTool: mcp.NewTool("get_suprsend_tenants",
			mcp.WithDescription("Use this tool to get all tenants."),
			mcp.WithNumber("limit",
				mcp.Description("Number of tenants to get. Default is 100."),
			),
			mcp.WithString("workspace",
				mcp.Description(`SuprSend workspace to get the tenants from.`),
			),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		Handler: getAllTenantsHandler,
	}

	upsert_suprsend_tenant := &Tool{
		Name:        "tenants.upsert",
		Description: "Enables upserting tenant information",
		MCPTool: mcp.NewTool("upsert_suprsend_tenant",
			mcp.WithDescription(`Use this tool to upsert a new tenant or update an existing tenant's properties.`),
			mcp.WithString("tenant_id",
				mcp.Description(`The tenant_id of the tenant to upsert.`),
				mcp.Required(),
			),
			mcp.WithString("workspace",
				mcp.Description(`SuprSend workspace to get the tenant from.`),
			),
			mcp.WithObject("tenant_properties",
				mcp.Description("The properties to upsert for the tenant."),
				mcp.Properties(tenantPropertiesSchema),
			),
			mcp.WithDestructiveHintAnnotation(true),
		),
		Handler: upsertTenantHandler,
	}

	update_tenant_default_preference := &Tool{
		Name:        "tenants.update_preferences",
		Description: "Enables updating category preference for a tenant",
		MCPTool: mcp.NewTool("update_suprsend_tenant_default_preference",
			mcp.WithDescription("Use this tool to update default preference for a tenant."),
			mcp.WithString("tenant_id",
				mcp.Description("The tenant_id of the tenant to update."),
				mcp.Required(),
			),
			mcp.WithString("category",
				mcp.Description("category_slug of an category to update."),
				mcp.Required(),
			),
			mcp.WithString(
				"preference",
				mcp.Description("The preference to update for the tenant."),
				mcp.Required(),
				mcp.Enum(
					"opt_in",
					"opt_out",
					"cant_unsubscribe",
				),
			),
			mcp.WithBoolean("visible_to_subscriber",
				mcp.Description("Whether the category is visible to subscribers."),
				mcp.Required(),
			),
			mcp.WithArray("mandatory_channels",
				mcp.Description("The channels to make mandatory for the category."),
				mcp.WithStringItems(),
				mcp.Required(),
			),
			mcp.WithArray("blocked_channels",
				mcp.Description("The channels to block for the category."),
				mcp.WithStringItems(),
				mcp.Required(),
			),
			mcp.WithString("workspace",
				mcp.Description(`SuprSend workspace to update the tenant from.`),
			),
			mcp.WithDestructiveHintAnnotation(true),
		),
		Handler: updateCategoryPreferenceTenant,
	}

	get_tenant_default_preference := &Tool{
		Name:        "tenants.get_preferences",
		Description: "Enables querying default preference for a tenant",
		MCPTool: mcp.NewTool("get_tenant_default_preference",
			mcp.WithDescription("Use this tool to query default preference for a tenant."),
			mcp.WithString("tenant_id",
				mcp.Description("The tenant_id of the tenant to get the default preference from."),
				mcp.Required(),
			),
			mcp.WithString("workspace",
				mcp.Description(`SuprSend workspace to get the tenant from.`),
			),
			mcp.WithDestructiveHintAnnotation(true),
		),
		Handler: getDefaultPreferenceTenant,
	}
	tools := []*Tool{
		get_suprsend_tenant,
		get_suprsend_tenants,
		upsert_suprsend_tenant,
		update_tenant_default_preference,
		get_tenant_default_preference,
	}
	return tools
}

func init() {
	for _, t := range newTenantTools() {
		RegisterTool(t, "tenants")
	}
}

func buildSocialLinks(m map[string]any) *suprsend.TenantSocialLinks {
	if m == nil {
		return nil
	}
	s := &suprsend.TenantSocialLinks{}
	if v, ok := m["facebook"].(string); ok {
		s.Facebook = &v
	}
	if v, ok := m["twitter"].(string); ok {
		s.Twitter = &v
	}
	if v, ok := m["instagram"].(string); ok {
		s.Instagram = &v
	}
	return s
}

var tenantPropertiesSchema = map[string]any{
	"tenant_name":              utils.StringSchema("Name of the tenant"),
	"logo":                     utils.StringSchema("Url of tenant's logo"),
	"timezone":                 utils.StringSchema("Timezone of the tenant"),
	"blocked_channels":         utils.StringSchema("Blocked channels of the tenant"),
	"embedded_preference_url":  utils.StringSchema("Embedded preference URL"),
	"hosted_preference_domain": utils.StringSchema("Hosted preference domain"),
	"primary_color":            utils.StringSchema("Primary color in hex format"),
	"secondary_color":          utils.StringSchema("Secondary color in hex format"),
	"tertiary_color":           utils.StringSchema("Tertiary color in hex format"),
	"social_links": map[string]any{
		"type":        "object",
		"description": "Social links of the tenant",
		"properties": map[string]any{
			"facebook":  utils.StringSchema("Facebook URL"),
			"twitter":   utils.StringSchema("Twitter URL"),
			"instagram": utils.StringSchema("Instagram URL"),
			"discord":   utils.StringSchema("Discord URL"),
			"telegram":  utils.StringSchema("Telegram URL"),
			"linkedin":  utils.StringSchema("Linkedin URL"),
			"medium":    utils.StringSchema("Medium URL"),
			"tiktok":    utils.StringSchema("Tiktok URL"),
			"website":   utils.StringSchema("Website URL"),
			"x":         utils.StringSchema("X URL"),
			"youtube":   utils.StringSchema("Youtube URL"),
		},
	},
	"custom_properties": map[string]any{
		"type":        "object",
		"description": "Custom properties of the tenant",
	},
}
