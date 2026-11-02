package mgmnt

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/suprsend/cli/internal/clierr"
	resty "resty.dev/v3"
)

// apiError converts an error HTTP response into a classified error.
// 401 → auth-invalid-token, 403 → auth-forbidden, all others → plain fmt.Errorf.
func apiError(resp *resty.Response) error {
	switch resp.StatusCode() {
	case http.StatusUnauthorized:
		msg := extractMessage(resp, "authentication failed: invalid or expired service token")
		return clierr.New(msg, clierr.CodeAuthInvalidToken).
			WithHint("Check that SUPRSEND_SERVICE_TOKEN or --service-token is correct")
	case http.StatusForbidden:
		msg := extractMessage(resp, "access denied: service token lacks permission for this resource")
		return clierr.New(msg, clierr.CodeAuthForbidden).
			WithHint("Ensure the service token has the required permissions")
	case http.StatusNotFound:
		msg := extractMessage(resp, "resource not found")
		if req := requestLabel(resp); req != "" {
			msg += " (" + req + ")"
		}
		return clierr.New(msg, clierr.CodeAPINotFound).
			WithHint("check the slug and the workspace (-w); `suprsend workspace list` shows the workspaces")
	}
	msg := extractMessage(resp, "")
	if msg != "" {
		return fmt.Errorf("request failed with message: %s", msg)
	}
	return fmt.Errorf("request failed: %s", resp.Status())
}

// CommitResult is the "commit_result" object that the workflow and schema push
// endpoints add when called with ?commit=true. They answer 201/202 even when
// the commit fails, so this is the only sign that nothing went live.
type CommitResult struct {
	IsCommitted bool  `json:"is_committed"`
	Errors      []any `json:"errors"`
}

// CommitStatus says what a request that could commit did.
type CommitStatus string

const (
	NotCommitted CommitStatus = ""          // no commit was requested, or it failed
	Committed    CommitStatus = "committed" // the draft went live
	NoChanges    CommitStatus = "unchanged" // the draft already matched live
)

// commitError turns a failed commit_result into a CLI error. "No uncommitted
// changes" means the draft already matches live, so it is not a failure.
func commitError(resource string, cr *CommitResult, code string) (CommitStatus, error) {
	if cr == nil || cr.IsCommitted {
		return Committed, nil
	}
	msgs := make([]string, 0, len(cr.Errors))
	for _, e := range cr.Errors {
		msgs = append(msgs, describeCommitError(fmt.Sprint(e)))
	}
	msg := strings.Join(msgs, "; ")
	if isNoChanges(msg) {
		log.Debugf("%s: no changes to commit", resource)
		return NoChanges, nil
	}
	return NotCommitted, clierr.New(fmt.Sprintf("%s: saved as draft but not committed: %s", resource, msg), code).
		WithResource(resource)
}

// bareFieldName matches a commit error that is only a quoted field name, such
// as 'variable_ns': the API's way of saying a required field is missing.
var bareFieldName = regexp.MustCompile(`^'([A-Za-z_][A-Za-z0-9_.]*)'$`)

func describeCommitError(msg string) string {
	if m := bareFieldName.FindStringSubmatch(strings.TrimSpace(msg)); m != nil {
		return fmt.Sprintf("required field %q is missing somewhere in the definition (the API doesn't say where)", m[1])
	}
	return msg
}

// requestError describes a request that got no HTTP response, even after
// retries. For a commit the outcome is unknown: the server may have committed
// before the connection dropped.
func requestError(resource string, commit bool, err error) error {
	if !commit {
		return fmt.Errorf("request failed: %w", err)
	}
	return clierr.New(fmt.Sprintf("%s: commit status unknown: request failed: %v", resource, err), clierr.CodeAPIInternal).
		WithResource(resource).
		WithHint("the commit may have gone through; run the same command again: it reports \"unchanged\" if it did")
}

// requestLabel returns "METHOD /path" for the request behind resp, so errors
// show which workspace and resource the API was asked about.
func requestLabel(resp *resty.Response) string {
	if resp == nil || resp.Request == nil || resp.Request.RawRequest == nil {
		return ""
	}
	return resp.Request.Method + " " + resp.Request.RawRequest.URL.Path
}

// The phrases the commit endpoints use when the draft already matches live
// (workflow/template, translation, preference categories).
var noChangesMarkers = []string{"no uncommitted changes", "no changes to commit", "no changes to publish"}

func isNoChanges(msg string) bool {
	msg = strings.ToLower(msg)
	for _, m := range noChangesMarkers {
		if strings.Contains(msg, m) {
			return true
		}
	}
	return false
}

// commitHTTPError is apiError for commit endpoints. They answer 400 or 409
// when there is nothing to commit; that is not a failure (a re-run of sync or
// commit should succeed), so it returns nil.
func commitHTTPError(resp *resty.Response, resource string) (CommitStatus, error) {
	code := resp.StatusCode()
	if (code == http.StatusBadRequest || code == http.StatusConflict) && isNoChanges(resp.String()) {
		log.Debugf("%s: no changes to commit", resource)
		return NoChanges, nil
	}
	return NotCommitted, apiError(resp)
}

func extractMessage(resp *resty.Response, fallback string) string {
	var e ErrorResponse
	if err := json.Unmarshal([]byte(resp.String()), &e); err == nil && e.Message != "" {
		return e.Message
	}
	return fallback
}
