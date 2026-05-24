package tools

import (
	"context"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/cli/pkg/mcpsdk"
	suprsend "github.com/suprsend/suprsend-go"
	"gopkg.in/yaml.v3"
)

func getTenantHandler(ctx context.Context, args mcpsdk.Args) (mcpsdk.Result, error) {
	tenant_id, err := args.RequireString("tenant_id")
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
	// todo: rename these
	user, err := suprsend_client.Tenants.Get(ctx, tenant_id)
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

func upsertTenantHandler(ctx context.Context, args mcpsdk.Args) (mcpsdk.Result, error) {
	tenant_id, err := args.RequireString("tenant_id")
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

	raw_props, ok := args.Map()["tenant_properties"]
	if !ok {
		return mcpsdk.Result{Text: "tenant_properties is required", IsError: true}, nil
	}

	tenant_properties, ok := raw_props.(map[string]any)
	if !ok {
		return mcpsdk.Result{Text: "invalid tenant_properties", IsError: true}, nil
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

	// NOTE: no naming collision with pkg/tenant today because this file does
	// not import it. If a future change adds that import, rename the local
	// `tenantResp` (or similar) to avoid shadowing the package name.
	tenant, err := suprsend_client.Tenants.Upsert(ctx, tenant_id, tenant_payload)
	if err != nil {
		if utils.IsAuthError(err) {
			markSessionDead(ctx)
		}
		err_str := err.Error()
		if strings.Contains(err_str, `{"tenant_name": "missing value"}`) {
			return mcpsdk.Result{Text: "tenant_name is required when creating a new tenant. Try again with a tenant_name.", IsError: true}, nil
		}
		return mcpsdk.Result{Text: err_str, IsError: true}, nil
	}

	yamluser, err := yaml.Marshal(tenant)
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}
	return mcpsdk.Result{Text: string(yamluser)}, nil
}

func updateCategoryPreferenceTenant(ctx context.Context, args mcpsdk.Args) (mcpsdk.Result, error) {
	tenantId, err := args.RequireString("tenant_id")
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}
	if tenantId == "" {
		return mcpsdk.Result{Text: "tenant_id is required", IsError: true}, nil
	}

	category, err := args.RequireString("category")
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}
	if category == "" {
		return mcpsdk.Result{Text: "category is required", IsError: true}, nil
	}

	rawArgs := args.Map()

	pref, ok := rawArgs["preference"].(string)
	if !ok {
		return mcpsdk.Result{Text: "preference must be a string", IsError: true}, nil
	}

	visibleToSubscriber, ok := rawArgs["visible_to_subscriber"].(bool)
	if !ok {
		return mcpsdk.Result{Text: "visible_to_subscriber must be bool", IsError: true}, nil
	}

	mandatoryChannelsAny, ok := rawArgs["mandatory_channels"]
	if !ok {
		mandatoryChannelsAny = []any{}
	}
	mandatoryChannelsSlice, ok := mandatoryChannelsAny.([]any)
	if !ok {
		return mcpsdk.Result{Text: "mandatory_channels must be an array", IsError: true}, nil
	}

	mandatoryChannels := make([]string, 0, len(mandatoryChannelsSlice))
	for _, v := range mandatoryChannelsSlice {
		s, ok := v.(string)
		if !ok {
			return mcpsdk.Result{Text: "mandatory_channels must be an array of strings", IsError: true}, nil
		}
		mandatoryChannels = append(mandatoryChannels, s)
	}

	blockedChannelsAny, ok := rawArgs["blocked_channels"]
	if !ok {
		blockedChannelsAny = []any{}
	}
	blockedChannelsSlice, ok := blockedChannelsAny.([]any)
	if !ok {
		return mcpsdk.Result{Text: "blocked_channels must be an array", IsError: true}, nil
	}

	blockedChannels := make([]string, 0, len(blockedChannelsSlice))
	for _, v := range blockedChannelsSlice {
		s, ok := v.(string)
		if !ok {
			return mcpsdk.Result{Text: "blocked_channels must be an array of strings", IsError: true}, nil
		}
		blockedChannels = append(blockedChannels, s)
	}

	prefPayload := suprsend.TenantCategoryPreferenceUpdateBody{
		Preference:          pref,
		VisibleToSubscriber: &visibleToSubscriber,
		MandatoryChannels:   mandatoryChannels,
		BlockedChannels:     blockedChannels,
	}

	workspace := args.GetString("workspace", "staging")

	suprsendClient, err := utils.GetSuprSendWorkspaceClient(workspace, ctx)
	if err != nil {
		if utils.IsAuthError(err) {
			markSessionDead(ctx)
		}
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	tenantPref, err := suprsendClient.Tenants.UpdateCategoryPreference(ctx, tenantId, category, prefPayload)
	if err != nil {
		if utils.IsAuthError(err) {
			markSessionDead(ctx)
		}
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	yamlPref, err := yaml.Marshal(tenantPref)
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	return mcpsdk.Result{Text: string(yamlPref)}, nil
}

func getDefaultPreferenceTenant(ctx context.Context, args mcpsdk.Args) (mcpsdk.Result, error) {
	tenantId, err := args.RequireString("tenant_id")
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

	tenantPref, err := suprsendClient.Tenants.GetAllCategoriesPreference(ctx, tenantId, nil)
	if err != nil {
		if utils.IsAuthError(err) {
			markSessionDead(ctx)
		}
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	yamlPref, err := yaml.Marshal(tenantPref)
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	return mcpsdk.Result{Text: string(yamlPref)}, nil
}

func getAllTenantsHandler(ctx context.Context, args mcpsdk.Args) (mcpsdk.Result, error) {
	workspace := args.GetString("workspace", "staging")
	limit := args.GetInt("limit", 100)

	suprsendClient, err := utils.GetSuprSendWorkspaceClient(workspace, ctx)
	if err != nil {
		if utils.IsAuthError(err) {
			markSessionDead(ctx)
		}
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	tenants, err := suprsendClient.Tenants.List(ctx, &suprsend.TenantListOptions{
		Limit: limit,
	})
	if err != nil {
		if utils.IsAuthError(err) {
			markSessionDead(ctx)
		}
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	yamlTenants, err := yaml.Marshal(tenants)
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}
	return mcpsdk.Result{Text: string(yamlTenants)}, nil
}

func newTenantTools() []*Tool {
	get_suprsend_tenant := &Tool{
		Tool: &mcpsdk.Tool{
			Name: "get_suprsend_tenant",
			Description: `Get a tenant's settings, branding metadata, and custom properties by tenant_id. Tenants are sub-accounts of a workspace, modeling end-customers in multi-tenant SaaS deployments.

When to use: the user references a tenant by id and you need its full state.

When NOT to use:
- To enumerate all tenants — use get_suprsend_tenants.
- For tenant-level preference defaults — use get_tenant_default_preference.

Returns: the tenant's settings (branding URLs, contact info, custom fields).`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"tenant_id": {Type: "string", Description: "The tenant_id of the tenant to get."},
					"workspace": {Type: "string", Description: "SuprSend workspace to get the tenant from."},
				},
				Required: []string{"tenant_id"},
			},
			Annotations: mcpsdk.Annotations{
				ReadOnlyHint:   true,
				IdempotentHint: true,
				OpenWorldHint:  true,
			},
			Handler: getTenantHandler,
		},
	}

	get_suprsend_tenants := &Tool{
		Tool: &mcpsdk.Tool{
			Name: "get_suprsend_tenants",
			Description: `List all tenants in the workspace. Use to discover tenant_ids before calling get_suprsend_tenant or upsert_suprsend_tenant.

Returns: up to limit tenants (default 100) with their id and properties.`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"limit":     {Type: "number", Description: "Number of tenants to get. Default is 100."},
					"workspace": {Type: "string", Description: "SuprSend workspace to get the tenants from."},
				},
			},
			Annotations: mcpsdk.Annotations{
				ReadOnlyHint:   true,
				IdempotentHint: true,
				OpenWorldHint:  true,
			},
			Handler: getAllTenantsHandler,
		},
	}

	upsert_suprsend_tenant := &Tool{
		Tool: &mcpsdk.Tool{
			Name: "upsert_suprsend_tenant",
			Description: `Create a new tenant or update an existing tenant's properties. Tenants are sub-accounts of a workspace, used to model end-customers in multi-tenant SaaS apps.

tenant_properties is merged with existing — not replaced. Pass only the fields you want to change. Common fields include name, branding URLs, and contact info.

When NOT to use:
- For preference defaults — use update_suprsend_tenant_default_preference.
- For users / objects within a tenant — use upsert_suprsend_user / upsert_suprsend_object.

Returns: the updated tenant on success.`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"tenant_id":         {Type: "string", Description: "The tenant_id of the tenant to upsert."},
					"workspace":         {Type: "string", Description: "SuprSend workspace to get the tenant from."},
					"tenant_properties": tenantPropertiesSchema,
				},
				Required: []string{"tenant_id"},
			},
			Annotations: mcpsdk.Annotations{
				DestructiveHint: true,
				IdempotentHint:  false,
				OpenWorldHint:   true,
			},
			Handler: upsertTenantHandler,
		},
	}

	update_tenant_default_preference := &Tool{
		Tool: &mcpsdk.Tool{
			Name: "update_suprsend_tenant_default_preference",
			Description: `Set the default category preference inherited by NEW users created in this tenant. Existing users are not affected; their preferences are independent.

preference values:
- opt_in — new users are opted into this category.
- opt_out — new users are opted out.
- cant_unsubscribe — new users are opted in AND locked from toggling.

mandatory_channels — channels users cannot disable for this category. blocked_channels — channels that cannot be enabled. visible_to_subscriber — whether end-users see this category in their preference UI.

When NOT to use:
- For per-user overrides — use update_suprsend_users_preferences.
- For per-object overrides — use update_suprsend_category_preference_object.

Side effects: changes apply only to users created AFTER this call. To retroactively update existing users, call update_suprsend_users_preferences per user.

Returns: the updated tenant default preference on success.`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"tenant_id": {Type: "string", Description: "The tenant_id of the tenant to update."},
					"category":  {Type: "string", Description: "category_slug of an category to update."},
					"preference": {
						Type:        "string",
						Description: "The preference to update for the tenant.",
						Enum:        []any{"opt_in", "opt_out", "cant_unsubscribe"},
					},
					"visible_to_subscriber": {Type: "boolean", Description: "Whether the category is visible to subscribers."},
					"mandatory_channels": {
						Type:        "array",
						Description: "The channels to make mandatory for the category.",
						Items:       &jsonschema.Schema{Type: "string"},
					},
					"blocked_channels": {
						Type:        "array",
						Description: "The channels to block for the category.",
						Items:       &jsonschema.Schema{Type: "string"},
					},
					"workspace": {Type: "string", Description: "SuprSend workspace to update the tenant from."},
				},
				Required: []string{"tenant_id", "category", "preference", "visible_to_subscriber", "mandatory_channels", "blocked_channels"},
			},
			Annotations: mcpsdk.Annotations{
				DestructiveHint: true,
				IdempotentHint:  true,
				OpenWorldHint:   true,
			},
			Handler: updateCategoryPreferenceTenant,
		},
	}

	get_tenant_default_preference := &Tool{
		Tool: &mcpsdk.Tool{
			Name: "get_tenant_default_preference",
			Description: `Read a tenant's default category preferences — the inheritance baseline applied to new users in this tenant.

When to use:
- Before update_suprsend_tenant_default_preference, to read current defaults.
- To diagnose why new users have unexpected preference state.

When NOT to use:
- For individual user / object preferences — use get_suprsend_user_preferences or get_suprsend_object_preferences.

Returns: the tenant's full default-preference tree.`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"tenant_id": {Type: "string", Description: "The tenant_id of the tenant to get the default preference from."},
					"workspace": {Type: "string", Description: "SuprSend workspace to get the tenant from."},
				},
				Required: []string{"tenant_id"},
			},
			Annotations: mcpsdk.Annotations{
				ReadOnlyHint:   true,
				IdempotentHint: true,
				OpenWorldHint:  true,
			},
			Handler: getDefaultPreferenceTenant,
		},
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

// tenantPropertiesSchema is the nested object schema for the tenant_properties
// argument on upsert_suprsend_tenant. Rewritten from a map[string]any literal
// to a typed *jsonschema.Schema so type errors surface at compile time
// (transformation reference, plan §3536-3544).
var tenantPropertiesSchema = &jsonschema.Schema{
	Type:        "object",
	Description: "The properties to upsert for the tenant.",
	Properties: map[string]*jsonschema.Schema{
		"tenant_name":              {Type: "string", Description: "Name of the tenant"},
		"logo":                     {Type: "string", Description: "Url of tenant's logo"},
		"timezone":                 {Type: "string", Description: "Timezone of the tenant"},
		"blocked_channels":         {Type: "string", Description: "Blocked channels of the tenant"},
		"embedded_preference_url":  {Type: "string", Description: "Embedded preference URL"},
		"hosted_preference_domain": {Type: "string", Description: "Hosted preference domain"},
		"primary_color":            {Type: "string", Description: "Primary color in hex format"},
		"secondary_color":          {Type: "string", Description: "Secondary color in hex format"},
		"tertiary_color":           {Type: "string", Description: "Tertiary color in hex format"},
		"social_links": {
			Type:        "object",
			Description: "Social links of the tenant",
			Properties: map[string]*jsonschema.Schema{
				"facebook":  {Type: "string", Description: "Facebook URL"},
				"twitter":   {Type: "string", Description: "Twitter URL"},
				"instagram": {Type: "string", Description: "Instagram URL"},
				"discord":   {Type: "string", Description: "Discord URL"},
				"telegram":  {Type: "string", Description: "Telegram URL"},
				"linkedin":  {Type: "string", Description: "Linkedin URL"},
				"medium":    {Type: "string", Description: "Medium URL"},
				"tiktok":    {Type: "string", Description: "Tiktok URL"},
				"website":   {Type: "string", Description: "Website URL"},
				"x":         {Type: "string", Description: "X URL"},
				"youtube":   {Type: "string", Description: "Youtube URL"},
			},
		},
		"custom_properties": {
			Type:        "object",
			Description: "Custom properties of the tenant",
		},
	},
}
