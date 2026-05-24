package mgmnt

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	log "github.com/sirupsen/logrus"
)

type Template struct {
	Slug            string   `json:"slug"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	Status          string   `json:"status"`
	Tags            []string `json:"tags"`
	EnabledChannels []string `json:"enabled_channels"`
}

type TemplateAPIResponse struct {
	Results []Template `json:"results"`
	Meta    struct {
		Count  int `json:"count"`
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	} `json:"meta"`
}

type TemplateVariantResponse struct {
	Results []map[string]any `json:"results"`
	Meta    struct {
		Count  int `json:"count"`
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	} `json:"meta"`
}

func (c *SS_MgmntClient) GetTemplateVariants(workspace, slug, mode string) ([]map[string]any, error) {
	if mode != "live" && mode != "draft" {
		return nil, fmt.Errorf("invalid mode: %s. Available modes are: live, draft", mode)
	}

	client := c.restyClient()
	defer client.Close()

	apiLimit := 50
	allVariants := []map[string]any{}
	currentOffset := 0
	for {
		urlStr := fmt.Sprintf("%sv2/%s/template/%s/variant/?mode=%s&include_content=true&limit=%d&offset=%d",
			c.mgmnt_base_URL, workspace, slug, mode, apiLimit, currentOffset)

		log.Debugf("Getting template variants for slug: %s, workspace: %s, mode: %s, limit: %d, offset: %d", slug, workspace, mode, apiLimit, currentOffset)
		resp, err := client.R().
			SetDebug(c.debug).
			SetHeader("Authorization", "ServiceToken "+c.serviceToken).
			SetResult(&TemplateVariantResponse{}).
			Get(urlStr)
		if err != nil {
			return nil, err
		}
		if resp.IsError() {
			return nil, apiError(resp)
		}

		page := resp.Result().(*TemplateVariantResponse)
		if len(page.Results) == 0 {
			break
		}
		allVariants = append(allVariants, page.Results...)
		currentOffset += len(page.Results)
		if len(page.Results) < apiLimit {
			break
		}
	}
	return allVariants, nil
}

func (c *SS_MgmntClient) CreateTemplate(workspace, slug string, enabledChannels []string) error {
	if slug == "" {
		return fmt.Errorf("slug cannot be empty")
	}

	client := c.restyClient()
	defer client.Close()

	urlStr := fmt.Sprintf("%sv2/%s/template/%s/", c.mgmnt_base_URL, workspace, slug)

	// The API is an upsert — POSTing to an existing slug updates it rather than returning 409.
	log.Debugf("Creating template %s in workspace %s", slug, workspace)
	resp, err := client.R().
		SetDebug(c.debug).
		SetHeader("Authorization", "ServiceToken "+c.serviceToken).
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]any{
			"enabled_channels": enabledChannels,
		}).
		Post(urlStr)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	if resp.IsError() {
		return apiError(resp)
	}
	return nil
}

func (c *SS_MgmntClient) PushTemplateVariant(workspace, slug string, variant map[string]any) error {
	channel, _ := variant["channel"].(string)
	variantID, _ := variant["id"].(string)
	if channel == "" || variantID == "" {
		return fmt.Errorf("variant is missing required 'channel' or 'id' field")
	}

	// Deep copy variant so we don't mutate the caller's map
	b, _ := json.Marshal(variant)
	var body map[string]any
	json.Unmarshal(b, &body)

	client := c.restyClient()
	defer client.Close()

	url := fmt.Sprintf("%sv2/%s/template/%s/channel/%s/variant/%s/", c.mgmnt_base_URL, workspace, slug, channel, variantID)

	log.Debugf("Pushing variant %s/%s for template %s in workspace %s", channel, variantID, slug, workspace)
	resp, err := client.R().
		SetDebug(c.debug).
		SetHeader("Authorization", "ServiceToken "+c.serviceToken).
		SetBody(body).
		Post(url)
	if err != nil {
		return err
	}
	if resp.IsError() {
		return apiError(resp)
	}
	return nil
}

type PreCommitVariant struct {
	Channel string                `json:"channel"`
	ID      string                `json:"id"`
	HasDiff bool                  `json:"has_diff"`
	Errors  map[string][]string   `json:"errors"`
}

type PreCommitValidateResponse struct {
	IsNew      bool               `json:"is_new"`
	HasChanges bool               `json:"has_changes"`
	Variants   []PreCommitVariant `json:"variants"`
}

func (c *SS_MgmntClient) PreCommitValidate(workspace, slug string) (*PreCommitValidateResponse, error) {
	if slug == "" {
		return nil, fmt.Errorf("slug cannot be empty")
	}

	client := c.restyClient()
	defer client.Close()

	urlStr := fmt.Sprintf("%sv2/%s/template/%s/pre_commit_validate/", c.mgmnt_base_URL, workspace, slug)

	log.Debugf("Pre-commit validating template %s in workspace %s", slug, workspace)
	resp, err := client.R().
		SetDebug(c.debug).
		SetHeader("Authorization", "ServiceToken "+c.serviceToken).
		SetHeader("Content-Type", "application/json").
		SetResult(&PreCommitValidateResponse{}).
		Post(urlStr)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	if resp.IsError() {
		return nil, apiError(resp)
	}
	return resp.Result().(*PreCommitValidateResponse), nil
}

func (c *SS_MgmntClient) CommitTemplate(workspace, slug, commitMessage string, variants []map[string]any) error {
	if slug == "" {
		return fmt.Errorf("slug cannot be empty")
	}

	client := c.restyClient()
	defer client.Close()

	urlEncodedCommitMessage := url.QueryEscape(commitMessage)
	urlStr := fmt.Sprintf("%sv2/%s/template/%s/commit/?commit_message=%s", c.mgmnt_base_URL, workspace, slug, urlEncodedCommitMessage)

	log.Debugf("Committing template %s in workspace %s", slug, workspace)
	req := client.R().
		SetDebug(c.debug).
		SetHeader("Authorization", "ServiceToken "+c.serviceToken).
		SetHeader("Content-Type", "application/json")

	if variants != nil {
		req.SetBody(map[string]any{
			"variants": variants,
		})
	}

	resp, err := req.Patch(urlStr)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	if resp.IsError() {
		return apiError(resp)
	}
	return nil
}

func (c *SS_MgmntClient) GetTemplateMockData(workspace, slug string) (map[string]any, error) {
	client := c.restyClient()
	defer client.Close()

	url := fmt.Sprintf("%sv2/%s/template/%s/mock_data/", c.mgmnt_base_URL, workspace, slug)

	log.Debugf("Getting mock data for template: %s, workspace: %s", slug, workspace)
	resp, err := client.R().
		SetDebug(c.debug).
		SetHeader("Authorization", "ServiceToken "+c.serviceToken).
		Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() == 404 {
		return nil, nil
	}
	if resp.IsError() {
		return nil, apiError(resp)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(resp.String()), &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	data, _ := result["data"].(map[string]any)
	return data, nil
}

func (c *SS_MgmntClient) PatchTemplateMockData(workspace, slug string, mockData map[string]any) error {
	client := c.restyClient()
	defer client.Close()

	url := fmt.Sprintf("%sv2/%s/template/%s/mock_data/", c.mgmnt_base_URL, workspace, slug)

	log.Debugf("Patching mock data for template: %s, workspace: %s", slug, workspace)
	resp, err := client.R().
		SetDebug(c.debug).
		SetHeader("Authorization", "ServiceToken "+c.serviceToken).
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]any{"data": mockData}).
		Patch(url)
	if err != nil {
		return err
	}
	if resp.IsError() {
		return apiError(resp)
	}
	return nil
}

func (c *SS_MgmntClient) GetTemplate(workspace, slug, mode string) (*Template, error) {
	if mode != "live" && mode != "draft" {
		return nil, fmt.Errorf("invalid mode: %s. Available modes are: live, draft", mode)
	}

	client := c.restyClient()
	defer client.Close()

	urlStr := fmt.Sprintf("%sv2/%s/template/%s/?mode=%s", c.mgmnt_base_URL, workspace, slug, mode)

	log.Debugf("Getting template: %s, workspace: %s, mode: %s", slug, workspace, mode)
	resp, err := client.R().
		SetDebug(c.debug).
		SetHeader("Authorization", "ServiceToken "+c.serviceToken).
		SetResult(&Template{}).
		Get(urlStr)
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, apiError(resp)
	}
	return resp.Result().(*Template), nil
}

func (c *SS_MgmntClient) ListTemplates(workspace string, limit int, offset int, mode string) (*TemplateAPIResponse, error) {
	if mode != "live" && mode != "draft" {
		return nil, fmt.Errorf("invalid mode: %s. Available modes are: live, draft", mode)
	}

	client := c.restyClient()
	defer client.Close()

	apiLimit := 50
	allTemplates := []Template{}
	currentOffset := offset
	remainingLimit := limit
	for remainingLimit > 0 {
		currentLimit := apiLimit
		if remainingLimit < apiLimit {
			currentLimit = remainingLimit
		}

		log.Debugf("Getting templates for workspace: %s, limit: %d, offset: %d", workspace, currentLimit, currentOffset)
		res, err := client.R().
			SetDebug(c.debug).
			SetHeader("Authorization", "ServiceToken "+c.serviceToken).
			SetResult(&TemplateAPIResponse{}).
			Get(c.mgmnt_base_URL + "v2/" + workspace + "/template/?limit=" + strconv.Itoa(currentLimit) + "&offset=" + strconv.Itoa(currentOffset) + "&mode=" + mode)
		if err != nil {
			log.Errorf("Error getting templates: %s", err)
			return nil, err
		}
		if res.IsError() {
			return nil, apiError(res)
		}

		templates := res.Result().(*TemplateAPIResponse)
		if len(templates.Results) == 0 {
			break
		}

		allTemplates = append(allTemplates, templates.Results...)
		remainingLimit -= len(templates.Results)
		currentOffset += len(templates.Results)
		if len(templates.Results) < currentLimit {
			break
		}
	}

	return &TemplateAPIResponse{
		Results: allTemplates,
		Meta: struct {
			Count  int `json:"count"`
			Limit  int `json:"limit"`
			Offset int `json:"offset"`
		}{
			Count:  len(allTemplates),
			Limit:  limit,
			Offset: currentOffset,
		},
	}, nil
}

type VariantOrderTenant struct {
	TenantID *string  `json:"tenant_id"`
	Variants []string `json:"variants"`
}

type VariantOrderChannel struct {
	Channel string               `json:"channel"`
	Tenants []VariantOrderTenant `json:"tenants"`
}

type VariantOrderResponse struct {
	Channels []VariantOrderChannel `json:"channels"`
}

func (c *SS_MgmntClient) GetVariantOrder(workspace, slug, mode string) (*VariantOrderResponse, error) {
	if mode != "live" && mode != "draft" {
		return nil, fmt.Errorf("invalid mode: %s. Available modes are: live, draft", mode)
	}

	client := c.restyClient()
	defer client.Close()

	urlStr := fmt.Sprintf("%sv2/%s/template/%s/variant/order/?mode=%s", c.mgmnt_base_URL, workspace, slug, mode)

	log.Debugf("Getting variant order for template: %s, workspace: %s, mode: %s", slug, workspace, mode)
	resp, err := client.R().
		SetDebug(c.debug).
		SetHeader("Authorization", "ServiceToken "+c.serviceToken).
		SetResult(&VariantOrderResponse{}).
		Get(urlStr)
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, apiError(resp)
	}
	return resp.Result().(*VariantOrderResponse), nil
}

func (c *SS_MgmntClient) PostVariantOrder(workspace, slug, mode string, order *VariantOrderResponse) error {
	if mode != "live" && mode != "draft" {
		return fmt.Errorf("invalid mode: %s. Available modes are: live, draft", mode)
	}

	client := c.restyClient()
	defer client.Close()

	urlStr := fmt.Sprintf("%sv2/%s/template/%s/variant/order/?mode=%s", c.mgmnt_base_URL, workspace, slug, mode)

	log.Debugf("Posting variant order for template: %s, workspace: %s, mode: %s", slug, workspace, mode)
	resp, err := client.R().
		SetDebug(c.debug).
		SetHeader("Authorization", "ServiceToken "+c.serviceToken).
		SetHeader("Content-Type", "application/json").
		SetBody(order).
		Post(urlStr)
	if err != nil {
		return err
	}
	if resp.IsError() {
		return apiError(resp)
	}
	return nil
}
