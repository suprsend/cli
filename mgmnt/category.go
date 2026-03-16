package mgmnt

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/suprsend/cli/internal/client"
)

type PreferenceCategoryResponse struct {
	RootCategories []RootCategory `json:"root_categories"`
	Hash           string         `json:"hash"`
	VersionNo      *int           `json:"version_no"`
	Status         string         `json:"status"`
	CommitMsg      string         `json:"commit_message"`
	CommittedAt    time.Time      `json:"committed_at"`
}

type RootCategory struct {
	RootCategory string    `json:"root_category"`
	Sections     []Section `json:"sections"`
}

type Section struct {
	Name              string     `json:"name"`
	Description       string     `json:"description"`
	Tags              []string   `json:"tags"`
	Categories        []Category `json:"categories"`
	MandatoryChannels []string   `json:"mandatory_channels"`
	DefaultPreference string     `json:"default_preference"`
}

type Category struct {
	Category                 string   `json:"category"`
	Name                     string   `json:"name"`
	Description              string   `json:"description"`
	DefaultPreference        string   `json:"default_preference"`
	DefaultMandatoryChannels []string `json:"default_mandatory_channels"`
	Tags                     []string `json:"tags"`
}

type CategoryPushResponse struct {
	ValidationResult struct {
		IsValid bool     `json:"is_valid"`
		Errors  []string `json:"errors"`
	} `json:"validation_result"`
}

type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (c *SS_MgmntClient) ListCategories(workspace, mode string) (*PreferenceCategoryResponse, error) {
	if mode != "live" && mode != "draft" {
		return nil, fmt.Errorf("invalid mode: %s. Available modes are: live, draft", mode)
	}

	client := client.NewHTTPClient()
	defer client.Close()
	urlStr, err := url.JoinPath(c.mgmnt_base_URL, "v1", workspace, "preference_category", "/")
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
		SetHeader("Content-Type", "application/json").
		SetResult(PreferenceCategoryResponse{}).
		Get(urlStr)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.IsError() {
		var errorResp ErrorResponse
		if err := json.Unmarshal([]byte(resp.String()), &errorResp); err == nil {
			return nil, fmt.Errorf("request failed with message: %s", errorResp.Message)
		}
		return nil, fmt.Errorf("request failed with status: %s", resp.Status())
	}

	result := resp.Result().(*PreferenceCategoryResponse)
	return result, nil
}

func (c *SS_MgmntClient) PushCategories(workspace string, categories interface{}, commit, commitMessage string) error {
	client := client.NewHTTPClient()
	defer client.Close()
	urlStr, err := url.JoinPath(c.mgmnt_base_URL, "v1", workspace, "preference_category", "/")
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
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
		SetBody(categories).
		SetResult(&CategoryPushResponse{}).
		Post(urlStr)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	if resp.IsError() {
		var errorResp ErrorResponse
		if err := json.Unmarshal([]byte(resp.String()), &errorResp); err == nil {
			return fmt.Errorf("request failed with message: %s", errorResp.Message)
		}
		return fmt.Errorf("request failed with status: %s", resp.Status())
	}
	if commit == "true" {
		result := resp.Result().(*CategoryPushResponse)
		if !result.ValidationResult.IsValid {
			fmt.Fprintf(os.Stdout, "Warning: validation failed: %v\n", result.ValidationResult.Errors)
		}
	}
	return nil
}

func (c *SS_MgmntClient) FinalizeCategories(workspace string, commitMessage string) error {
	client := client.NewHTTPClient()
	defer client.Close()
	urlStr, err := url.JoinPath(c.mgmnt_base_URL, "v1", workspace, "preference_category", "commit", "/")
	if err != nil {
		return fmt.Errorf("failed constructing url: %w", err)
	}
	u, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("failed parsing url: %w", err)
	}
	q := u.Query()
	q.Add("commit_message", commitMessage)
	u.RawQuery = q.Encode()
	urlStr = u.String()
	resp, err := client.R().
		SetDebug(c.debug).
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", "ServiceToken "+c.serviceToken).
		Patch(urlStr)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	if resp.IsError() {
		var errorResp ErrorResponse
		if err := json.Unmarshal([]byte(resp.String()), &errorResp); err == nil {
			return fmt.Errorf("request failed with message: %s", errorResp.Message)
		}
	}
	return nil
}
