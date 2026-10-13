package mgmnt

import (
	"encoding/json"
	"fmt"
	"net/http"
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
		return clierr.New(msg, clierr.CodeAPINotFound)
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

// commitError turns a failed commit_result into a CLI error. "No uncommitted
// changes" means the draft already matches live, so it is not a failure.
func commitError(resource string, cr *CommitResult, code string) error {
	if cr == nil || cr.IsCommitted {
		return nil
	}
	msgs := make([]string, 0, len(cr.Errors))
	for _, e := range cr.Errors {
		msgs = append(msgs, fmt.Sprint(e))
	}
	msg := strings.Join(msgs, "; ")
	if isNoChanges(msg) {
		log.Infof("%s: no changes to commit", resource)
		return nil
	}
	return clierr.New(fmt.Sprintf("%s: saved as draft but not committed: %s", resource, msg), code).
		WithResource(resource)
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
func commitHTTPError(resp *resty.Response, resource string) error {
	code := resp.StatusCode()
	if (code == http.StatusBadRequest || code == http.StatusConflict) && isNoChanges(resp.String()) {
		log.Infof("%s: no changes to commit", resource)
		return nil
	}
	return apiError(resp)
}

func extractMessage(resp *resty.Response, fallback string) string {
	var e ErrorResponse
	if err := json.Unmarshal([]byte(resp.String()), &e); err == nil && e.Message != "" {
		return e.Message
	}
	return fallback
}
