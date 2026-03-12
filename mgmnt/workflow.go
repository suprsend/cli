package mgmnt

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/suprsend/cli/internal/client"
)

type Workflow struct {
	Slug      string   `json:"slug"`
	IsEnabled bool     `json:"is_enabled"`
	Status    string   `json:"status"`
	Category  string   `json:"category"`
	Tags      []string `json:"tags"`
}

type WorkflowPushResponse struct {
	ValidationResult struct {
		IsValid bool     `json:"is_valid"`
		Errors  []string `json:"errors"`
	} `json:"validation_result"`
}

type WorkflowAPIResponse struct {
	Results []Workflow `json:"results"`
	Meta    struct {
		Count  int `json:"count"`
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	} `json:"meta"`
}

type WorkflowsResponse struct {
	Results []any `json:"results"`
	Meta    struct {
		Count  int `json:"count"`
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	} `json:"meta"`
}

type WorkflowDetailResponse struct {
	Slug           string `json:"slug"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	Status         string `json:"status"`
	Category       string `json:"category"`
	CommitMessage  string `json:"commit_message"`
	LastExecutedAt string `json:"last_executed_at"`
}

func (c *SS_MgmntClient) ListWorkflows(workspace string, limit int, offset int, mode string) (*WorkflowAPIResponse, error) {
	if mode != "live" && mode != "draft" {
		return nil, fmt.Errorf("invalid mode: %s. Available modes are: live, draft", mode)
	}

	client := client.NewHTTPClient()
	defer client.Close()

	apiLimit := 50
	allWorkflows := []Workflow{}
	currentOffset := offset
	remainingLimit := limit
	for remainingLimit > 0 {
		currentLimit := apiLimit
		if remainingLimit < apiLimit {
			currentLimit = remainingLimit
		}

		log.Debugf("Getting workflows for workspace: %s, limit: %d, offset: %d", workspace, currentLimit, currentOffset)
		urlStr, err := url.JoinPath(c.mgmnt_base_URL, "v1", workspace, "workflow", "/")
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
		res, err := client.R().
			SetDebug(c.debug).
			SetHeader("Authorization", "ServiceToken "+c.serviceToken).
			SetResult(&WorkflowAPIResponse{}).
			Get(urlStr)
		if err != nil {
			log.Errorf("Error getting workflows: %s", err)
			return nil, err
		}
		if res.IsError() {
			var errorResp ErrorResponse
			if err := json.Unmarshal([]byte(res.String()), &errorResp); err == nil {
				return nil, fmt.Errorf("request failed with message: %s", errorResp.Message)
			}
			return nil, fmt.Errorf("request failed: %s", res.Status())
		}

		workflows := res.Result().(*WorkflowAPIResponse)
		if len(workflows.Results) == 0 {
			break
		}

		allWorkflows = append(allWorkflows, workflows.Results...)
		remainingLimit -= len(workflows.Results)
		currentOffset += len(workflows.Results)
		if len(workflows.Results) < currentLimit {
			break
		}
	}

	return &WorkflowAPIResponse{
		Results: allWorkflows,
		Meta: struct {
			Count  int `json:"count"`
			Limit  int `json:"limit"`
			Offset int `json:"offset"`
		}{
			Count:  len(allWorkflows),
			Limit:  limit,
			Offset: currentOffset,
		},
	}, nil
}

func (c *SS_MgmntClient) GetWorkflowDetailBySlug(workspace, slug, mode string) (*map[string]any, error) {
	client := client.NewHTTPClient()
	defer client.Close()
	urlStr, err := url.JoinPath(c.mgmnt_base_URL, "v1", workspace, "workflow", slug, "/")
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
		return nil, err
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

func (c *SS_MgmntClient) GetWorkflowDetail(workspace, slug, mode string) (*WorkflowDetailResponse, error) {
	client := client.NewHTTPClient()
	defer client.Close()

	urlStr, err := url.JoinPath(c.mgmnt_base_URL, "v1", workspace, "workflow", slug, "/")
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
		SetResult(&WorkflowDetailResponse{}).
		Get(urlStr)
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		var errorResp ErrorResponse
		if err := json.Unmarshal([]byte(resp.String()), &errorResp); err == nil {
			return nil, fmt.Errorf("request failed with message: %s", errorResp.Message)
		}
		return nil, fmt.Errorf("request failed: %s", resp.Status())
	}

	workflowResp := resp.Result().(*WorkflowDetailResponse)
	return workflowResp, nil
}

func (c *SS_MgmntClient) GetWorkflows(workspace, mode string) (*WorkflowsResponse, error) {
	if mode != "live" && mode != "draft" {
		return nil, fmt.Errorf("invalid mode: %s, Available modes are: live, draft", mode)
	}

	client := client.NewHTTPClient()
	defer client.Close()

	limit := 50
	offset := 0
	allWorkflows := []any{}
	totalCount := 0

	for {
		urlStr, err := url.JoinPath(c.mgmnt_base_URL, "v1", workspace, "workflow", "/")
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
			SetResult(&WorkflowsResponse{}).
			Get(urlStr)
		if err != nil {
			fmt.Fprintf(os.Stdout, "Error: Failed to get workflows: %v\n", err)
			return nil, err
		}

		if res.IsError() {
			var errorResp ErrorResponse
			if err := json.Unmarshal([]byte(res.String()), &errorResp); err == nil {
				return nil, fmt.Errorf("request failed with message: %s", errorResp.Message)
			}
			return nil, fmt.Errorf("request failed: %s", res.Status())
		}

		workflows := res.Result().(*WorkflowsResponse)

		if len(workflows.Results) == 0 {
			break
		}

		allWorkflows = append(allWorkflows, workflows.Results...)
		totalCount += len(workflows.Results)
		offset += limit
	}

	return &WorkflowsResponse{
		Results: allWorkflows,
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

func (c *SS_MgmntClient) PushWorkflow(workspace, slug string, workflow map[string]any, commit, commitMessage string) error {
	if slug == "" {
		return fmt.Errorf("slug cannot be empty")
	}

	client := client.NewHTTPClient()
	defer client.Close()

	urlStr, err := url.JoinPath(c.mgmnt_base_URL, "v1", workspace, "workflow", slug, "/")
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
	log.Debugf("Pushing workflow to: %s", urlStr)

	res, err := client.R().
		SetDebug(c.debug).
		SetHeader("Authorization", "ServiceToken "+c.serviceToken).
		SetHeader("Content-Type", "application/json").
		SetBody(workflow).
		SetResult(&WorkflowPushResponse{}).
		Post(urlStr)
	if err != nil {
		log.Errorf("Error pushing workflow: %s", err)
		return err
	}
	if res.IsError() {
		var errorResp ErrorResponse
		if err := json.Unmarshal([]byte(res.String()), &errorResp); err == nil {
			return fmt.Errorf("request failed with message: %s", errorResp.Message)
		}
		return fmt.Errorf("request failed: %s", res.Status())
	}
	if commit == "true" {
		validationResult := res.Result().(*WorkflowPushResponse)
		if !validationResult.ValidationResult.IsValid {
			fmt.Fprintf(os.Stdout, "Warning: Workflow %s is not valid: %v\n", slug, validationResult.ValidationResult.Errors)
		}
	}
	return nil
}

func (c *SS_MgmntClient) ChangeStatusWorkflow(workspace, slug string, enabled bool) error {
	if slug == "" {
		return fmt.Errorf("slug cannot be empty")
	}

	client := client.NewHTTPClient()
	defer client.Close()
	urlStr, err := url.JoinPath(c.mgmnt_base_URL, "v1", workspace, "workflow", slug, "enable", "/")
	if err != nil {
		return fmt.Errorf("failed constructing url: %w", err)
	}
	u, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("failed parsing url: %w", err)
	}
	urlStr = u.String()
	fmt.Println(urlStr)
	fmt.Println(urlStr)
	body := map[string]interface{}{
		"is_enabled": enabled,
	}

	action := "disabling"
	if enabled {
		action = "enabling"
	}

	log.Debugf("Finalizing workflow (slug: %s) by %s", slug, action)

	res, err := client.R().
		SetDebug(c.debug).
		SetHeader("Authorization", "ServiceToken "+c.serviceToken).
		SetHeader("Content-Type", "application/json").
		SetBody(body).
		Patch(urlStr)
	if err != nil {
		return fmt.Errorf("network error: %w", err)
	}

	if res.IsError() {
		if res.StatusCode() == 404 {
			return fmt.Errorf("workflow not found: %s", slug)
		}
		return fmt.Errorf("%s failed: %s", action, res.Status())
	}

	return nil
}
