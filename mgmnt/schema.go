package mgmnt

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"

	"github.com/suprsend/cli/internal/client"
	"resty.dev/v3"
)

type SchemasResponse struct {
	Meta struct {
		Count  int `json:"count"`
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	} `json:"meta"`
	Results []any `json:"results"`
}

type ListSchemaResponse struct {
	Meta struct {
		Count  int `json:"count"`
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	} `json:"meta"`
	Results []SchemaResponse `json:"results"`
}

type SchemaResponse struct {
	Slug        string     `json:"slug"`
	Title       string     `json:"title"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	IsEnabled   bool       `json:"is_enabled"`
	JSONSchema  JSONSchema `json:"json_schema"`
}

type LinkedSchemasResponse struct {
	Results []LinkedSchemas `json:"results"`
	Meta    struct {
		Count  int `json:"count"`
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	} `json:"meta"`
}

type LinkedSchemas struct {
	Slug            string     `json:"slug"`
	VersionNo       *int       `json:"version_no"`
	Name            string     `json:"name"`
	Description     string     `json:"description"`
	JSONSchema      JSONSchema `json:"json_schema"`
	LinkedWorkflows []string   `json:"linked_workflows"`
	LinkedEvents    []string   `json:"linked_events"`
	CreatedAt       string     `json:"created_at"`
}

type SchemaPayload struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	IsEnabled   string     `json:"is_enabled"`
	JSONSchema  JSONSchema `json:"json_schema"`
}

type JSONSchema struct {
	Type       string                 `json:"type"`
	Defs       map[string]interface{} `json:"$defs"`
	Title      string                 `json:"title"`
	Required   *[]string              `json:"required,omitempty"`
	Properties map[string]interface{} `json:"properties"`
}

type Property struct {
	Type *string `json:"type,omitempty"`
	Ref  *string `json:"$ref,omitempty"`
}

func (c *SS_MgmntClient) ListSchema(workspace string, limit, offset int, mode string) (*ListSchemaResponse, error) {
	if mode != "live" && mode != "draft" {
		return nil, fmt.Errorf("invalid mode: %s. Available modes are: live, draft", mode)
	}
	client := client.NewHTTPClient()
	defer client.Close()

	apiLimit := 50
	allSchemas := []SchemaResponse{}
	currentOffset := offset
	remainingLimit := limit

	for remainingLimit > 0 {
		currentLimit := apiLimit
		if remainingLimit < apiLimit {
			currentLimit = remainingLimit
		}

		urlStr, err := url.JoinPath(c.mgmnt_base_URL, "v1", workspace, "schema", "/")
		if err != nil {
			return nil, fmt.Errorf("failed constructing url: %w", err)
		}
		u, err := url.Parse(urlStr)
		if err != nil {
			return nil, fmt.Errorf("failed parsing url: %w", err)
		}
		q := u.Query()
		q.Add("limit", strconv.Itoa(currentLimit))
		q.Add("offset", strconv.Itoa(currentOffset))
		q.Add("mode", mode)
		u.RawQuery = q.Encode()
		urlStr = u.String()
		resp, err := client.R().
			SetDebug(c.debug).
			SetHeader("Authorization", "ServiceToken "+c.serviceToken).
			SetHeader("Content-Type", "application/json").
			SetResult(&ListSchemaResponse{}).
			Get(urlStr)
		if err != nil {
			return nil, fmt.Errorf("request failed: %w", err)
		}
		if resp.IsError() {
			var errorResp ErrorResponse
			if err := json.Unmarshal([]byte(resp.String()), &errorResp); err == nil {
				return nil, fmt.Errorf("request failed with message: %s", errorResp.Message)
			}
		}

		schemas := resp.Result().(*ListSchemaResponse)
		if len(schemas.Results) == 0 {
			break
		}

		allSchemas = append(allSchemas, schemas.Results...)
		remainingLimit -= len(schemas.Results)
		currentOffset += len(schemas.Results)
		if len(schemas.Results) < currentLimit {
			break
		}
	}

	return &ListSchemaResponse{
		Results: allSchemas,
		Meta: struct {
			Count  int `json:"count"`
			Limit  int `json:"limit"`
			Offset int `json:"offset"`
		}{
			Count:  len(allSchemas),
			Limit:  limit,
			Offset: offset,
		},
	}, nil
}

func (c *SS_MgmntClient) GetSchema(workspace, slug string, version string) (*SchemaResponse, error) {
	client := client.NewHTTPClient()
	defer client.Close()
	urlStr, err := url.JoinPath(c.mgmnt_base_URL, "v1", workspace, "schema", slug, "/")
	if err != nil {
		return nil, fmt.Errorf("failed constructing url: %w", err)
	}
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed parsing url: %w", err)
	}
	q := u.Query()
	q.Add("version", version)
	u.RawQuery = q.Encode()
	urlStr = u.String()
	res, err := client.R().
		SetDebug(c.debug).
		SetHeader("Authorization", "ServiceToken "+c.serviceToken).
		SetResult(&SchemaResponse{}).
		Get(urlStr)
	if err != nil {
		return nil, fmt.Errorf("request failed: %s", err.Error())
	}
	if res.IsError() {
		var errorResp ErrorResponse
		if err := json.Unmarshal([]byte(res.String()), &errorResp); err == nil {
			return nil, fmt.Errorf("request failed with message: %s", errorResp.Message)
		}
		return nil, fmt.Errorf("request failed: %s", res.Status())
	}
	schema := res.Result().(*SchemaResponse)
	if schema.JSONSchema.Properties == nil {
		return nil, fmt.Errorf("schema properties are empty")
	}
	return schema, nil
}

func (c *SS_MgmntClient) GetSchemaBySlug(workspace, slug, mode string) (*map[string]any, error) {
	if mode != "live" && mode != "draft" {
		return nil, fmt.Errorf("invalid mode: %s. Available modes are: live, draft", mode)
	}
	client := client.NewHTTPClient()
	defer client.Close()
	urlStr, err := url.JoinPath(c.mgmnt_base_URL, "v1", workspace, "schema", slug, "/")
	if err != nil {
		return nil, fmt.Errorf("failed constructing url: %w", err)
	}
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed parsing url: %w", err)
	}
	q := u.Query()
	q.Add("mode", mode)
	u.RawQuery = q.Encode()
	urlStr = u.String()
	resp, err := client.R().
		SetDebug(c.debug).
		SetHeader("Authorization", "ServiceToken "+c.serviceToken).
		SetResult(&map[string]any{}).
		Get(urlStr)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	if resp.IsError() {
		var errorResp ErrorResponse
		if err := json.Unmarshal([]byte(resp.String()), &errorResp); err == nil {
			return nil, fmt.Errorf("request failed with message: %s", errorResp.Message)
		}
		return nil, fmt.Errorf("request failed: %s", resp.Status())
	}

	return resp.Result().(*map[string]any), nil
}

func (c *SS_MgmntClient) GetLinkedSchemas(workspace, mode string) (*LinkedSchemasResponse, error) {
	if mode != "live" && mode != "draft" {
		return nil, fmt.Errorf("invalid mode: %s, Available modes are: live, draft", mode)
	}

	client := client.NewHTTPClient()
	defer client.Close()

	limit := 50
	offset := 0
	allSchemas := []LinkedSchemas{}
	totalCount := 0

	for {
		res, err := client.R().
			SetDebug(c.debug).
			SetHeader("Authorization", "ServiceToken "+c.serviceToken).
			SetResult(&LinkedSchemasResponse{}).
			Get(c.mgmnt_base_URL + "v1/" + workspace + "/schema/all/linked/?limit=" + strconv.Itoa(limit) + "&offset=" + strconv.Itoa(offset) + "&mode=" + mode)
		if err != nil {
			fmt.Fprintf(os.Stdout, "Error: Failed to get schemas: %v\n", err)
			return nil, err
		}
		if res.IsError() {
			var errorResp ErrorResponse
			if err := json.Unmarshal([]byte(res.String()), &errorResp); err == nil {
				return nil, fmt.Errorf("request failed with message: %s", errorResp.Message)
			}
			return nil, fmt.Errorf("request failed: %s", res.Status())
		}

		schemas := res.Result().(*LinkedSchemasResponse)

		if len(schemas.Results) == 0 {
			break
		}

		allSchemas = append(allSchemas, schemas.Results...)
		totalCount += len(schemas.Results)
		offset += limit
	}

	return &LinkedSchemasResponse{
		Results: allSchemas,
		Meta: struct {
			Count  int `json:"count"`
			Limit  int `json:"limit"`
			Offset int `json:"offset"`
		}{
			Count:  totalCount,
			Limit:  limit,
			Offset: 0,
		},
	}, nil
}

func (c *SS_MgmntClient) GetSchemas(workspace, mode string) (*SchemasResponse, error) {
	if mode != "live" && mode != "draft" {
		return nil, fmt.Errorf("invalid mode: %s, Available modes are: live, draft", mode)
	}

	client := client.NewHTTPClient()
	defer client.Close()

	limit := 50
	offset := 0
	allSchemas := []any{}
	totalCount := 0

	for {
		urlStr, err := url.JoinPath(c.mgmnt_base_URL, "v1", workspace, "schema", "/")
		if err != nil {
			return nil, fmt.Errorf("failed constructing url: %w", err)
		}
		u, err := url.Parse(urlStr)
		if err != nil {
			return nil, fmt.Errorf("failed parsing url: %w", err)
		}
		q := u.Query()
		q.Add("limit", strconv.Itoa(limit))
		q.Add("offset", strconv.Itoa(offset))
		q.Add("mode", mode)
		u.RawQuery = q.Encode()
		urlStr = u.String()
		res, err := client.R().
			SetDebug(c.debug).
			SetHeader("Authorization", "ServiceToken "+c.serviceToken).
			SetResult(&SchemasResponse{}).
			Get(urlStr)
		if err != nil {
			fmt.Fprintf(os.Stdout, "Error: Failed to get schemas: %v\n", err)
			return nil, err
		}
		if res.IsError() {
			var errorResp ErrorResponse
			if err := json.Unmarshal([]byte(res.String()), &errorResp); err == nil {
				return nil, fmt.Errorf("request failed with message: %s", errorResp.Message)
			}
			return nil, fmt.Errorf("request failed: %s", res.Status())
		}

		schemas := res.Result().(*SchemasResponse)

		if len(schemas.Results) == 0 {
			break
		}

		allSchemas = append(allSchemas, schemas.Results...)
		totalCount += len(schemas.Results)
		offset += limit
	}

	return &SchemasResponse{
		Results: allSchemas,
		Meta: struct {
			Count  int `json:"count"`
			Limit  int `json:"limit"`
			Offset int `json:"offset"`
		}{
			Count:  totalCount,
			Limit:  limit,
			Offset: 0,
		},
	}, nil
}

func (c *SS_MgmntClient) PushSchema(workspace, schemaSlug string, payload map[string]any, commit, commitMessage string) error {
	client := client.NewHTTPClient()
	defer client.Close()
	urlStr, err := url.JoinPath(c.mgmnt_base_URL, "v1", workspace, "schema", schemaSlug, "/")
	if err != nil {
		return fmt.Errorf("failed constructing url: %w", err)
	}
	u, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("failed parsing url: %w", err)
	}
	q := u.Query()
	q.Add("commit", commit)
	q.Add("commit_message", commitMessage)
	u.RawQuery = q.Encode()
	urlStr = u.String()

	resp, err := client.R().
		SetDebug(c.debug).
		SetHeader("Authorization", "ServiceToken "+c.serviceToken).
		SetHeader("Content-Type", "application/json").
		SetBody(payload).
		Post(urlStr)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	if resp.IsError() {
		var errorResp ErrorResponse
		if err := json.Unmarshal([]byte(resp.String()), &errorResp); err == nil {
			return fmt.Errorf("request failed with message: %s", errorResp.Message)
		}
		return fmt.Errorf("request failed: %s", resp.Status())
	}
	return nil
}

func (c *SS_MgmntClient) FinalizeSchema(workspace, slug, commitMessage string) error {
	if slug == "" {
		return fmt.Errorf("slug cannot be empty")
	}
	client := resty.New()
	defer client.Close()

	urlStr, err := url.JoinPath(c.mgmnt_base_URL, "v1", workspace, "schema", slug, "commit", "/")
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	u, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("failed parsing url: %w", err)
	}
	q := u.Query()
	q.Add("commit_message", commitMessage)
	u.RawQuery = q.Encode()
	urlStr = u.String()
	res, err := client.R().
		SetDebug(c.debug).
		SetHeader("Authorization", "ServiceToken "+c.serviceToken).
		SetHeader("Content-Type", "application/json").
		Patch(urlStr)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	if res.IsError() {
		var errorResp ErrorResponse
		if err := json.Unmarshal([]byte(res.String()), &errorResp); err == nil {
			return fmt.Errorf("request failed with message: %s", errorResp.Message)
		}
		return fmt.Errorf("request failed: %s", res.Status())
	}
	return nil
}
