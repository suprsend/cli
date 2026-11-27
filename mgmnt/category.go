package mgmnt

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/suprsend/cli/internal/client"
	"github.com/suprsend/cli/internal/clierr"
	resty "resty.dev/v3"
)

type PreferenceCategoryResponse struct {
	Schema         string         `json:"$schema,omitempty"`
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
	Name string `json:"name"`
	// Slug identifies the section (its translations are keyed by it) and
	// stays the same when the name changes. Without it the API derives the
	// slug from the name again, so it must round-trip through the file. A
	// nameless section, the bucket for categories directly under a root,
	// has none.
	Slug              string     `json:"slug,omitempty"`
	Description       string     `json:"description"`
	Tags              []string   `json:"tags"`
	Categories        []Category `json:"categories"`
	MandatoryChannels []string   `json:"mandatory_channels"`
	DefaultPreference string     `json:"default_preference"`
}

type Category struct {
	// Category is the category's slug, its identity. Empty is omitted: the
	// API then derives it from Name, so a rename must still send the stored
	// slug.
	Category                 string   `json:"category,omitempty"`
	Name                     string   `json:"name"`
	Description              string   `json:"description"`
	DefaultPreference        string   `json:"default_preference"`
	DefaultMandatoryChannels []string `json:"default_mandatory_channels"`
	DefaultOptInChannels     []string `json:"default_opt_in_channels,omitempty"`
	Tags                     []string `json:"tags"`
	//
	DigestSchedule map[string]any   `json:"digest_schedule"`
	Properties     []map[string]any `json:"properties"`
	// IsCommercial controls one-click unsubscribe headers.
	// Nil keeps the server default: true for promotional categories, false for transactional and system.
	// System categories, and any category with default_preference=cant_unsubscribe, cannot be true.
	IsCommercial *bool `json:"is_commercial,omitempty"`
}

type CategoryPushResponse struct {
	ValidationResult struct {
		IsValid bool  `json:"is_valid"`
		Errors  []any `json:"errors"`
	} `json:"validation_result"`
	CommitResult *CommitResult `json:"commit_result"`
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
		return nil, apiError(resp)
	}

	result := resp.Result().(*PreferenceCategoryResponse)
	return result, nil
}

// staleDraftMarker is in the 409 the category push answers when the draft's
// hash is no longer expected_hash.
const staleDraftMarker = "draft has changed"

// PushCategories replaces the workspace's draft category tree, and with
// commit publishes it in the same transaction. expectedHash, when set, is
// the hash of the draft the caller compared with: the API then refuses the
// push (api-conflict) if the draft has changed since. With commit, a tree
// that already matches live and a failed pre-commit check both leave the
// draft as it was.
func (c *SS_MgmntClient) PushCategories(workspace string, categories interface{}, commit bool, commitMessage, expectedHash string) (CommitStatus, error) {
	client := client.NewHTTPClient()
	defer client.Close()
	urlStr, err := url.JoinPath(c.mgmnt_base_URL, "v1", workspace, "preference_category", "/")
	if err != nil {
		return NotCommitted, fmt.Errorf("request failed: %w", err)
	}
	u, err := url.Parse(urlStr)
	if err != nil {
		return NotCommitted, fmt.Errorf("failed parsing url: %w", err)
	}
	q := u.Query()
	q.Add("commit", strconv.FormatBool(commit))
	q.Add("commit_message", commitMessage)
	if expectedHash != "" {
		q.Add("expected_hash", expectedHash)
	}
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
		return NotCommitted, requestError("preference_categories", commit, err)
	}
	if resp.IsError() {
		// With commit, a tree that already matches live answers 409 "no
		// changes to publish"; a re-run of sync or push must not fail on it.
		if commit && resp.StatusCode() == http.StatusConflict && isNoChanges(resp.String()) {
			log.Debugf("preference_categories: no changes to commit")
			return NoChanges, nil
		}
		return NotCommitted, categoryPushError(resp, commit)
	}
	if commit {
		// Older versions of the service answer 201 even when the commit
		// fails, with the outcome in commit_result.
		result := resp.Result().(*CategoryPushResponse)
		if result.CommitResult != nil && !result.CommitResult.IsCommitted && !result.ValidationResult.IsValid {
			return NotCommitted, clierr.New(fmt.Sprintf("preference_categories: saved as draft but not committed: categories are invalid: %v", result.ValidationResult.Errors),
				clierr.CodeAPIValidation).WithResource("preference_categories")
		}
		return commitError("preference_categories", result.CommitResult, clierr.CodeAPIConflict, nil)
	}
	return NotCommitted, nil
}

// categoryPushError explains an error reply to a category push. Nothing was
// saved in any of these cases: the push and its commit are one transaction.
func categoryPushError(resp *resty.Response, commit bool) (err error) {
	const resource = "preference_categories"
	msg := extractMessage(resp, resp.Status())
	switch resp.StatusCode() {
	case http.StatusConflict:
		if strings.Contains(strings.ToLower(msg), staleDraftMarker) {
			return clierr.New(fmt.Sprintf("%s: the draft changed in the workspace while this push was running; nothing was saved", resource), clierr.CodeAPIConflict).
				WithResource(resource).
				WithHint("run the command again: it compares with the draft as it is now")
		}
	case http.StatusBadRequest:
		what := "rejected by the API"
		if commit {
			what = "not committed, and the draft was left as it was"
		}
		return clierr.New(fmt.Sprintf("%s: %s: %s", resource, what, msg), clierr.CodeAPIValidation).WithResource(resource)
	}
	return apiError(resp)
}

func (c *SS_MgmntClient) FinalizeCategories(workspace string, commitMessage string) (CommitStatus, error) {
	client := client.NewHTTPClient()
	defer client.Close()
	urlStr, err := url.JoinPath(c.mgmnt_base_URL, "v1", workspace, "preference_category", "commit", "/")
	if err != nil {
		return NotCommitted, fmt.Errorf("failed constructing url: %w", err)
	}
	u, err := url.Parse(urlStr)
	if err != nil {
		return NotCommitted, fmt.Errorf("failed parsing url: %w", err)
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
		return NotCommitted, requestError("preference_categories", true, err)
	}
	if resp.IsError() {
		return commitHTTPError(resp, "preference_categories")
	}
	return Committed, nil
}
