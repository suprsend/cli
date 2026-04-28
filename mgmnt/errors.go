package mgmnt

import (
	"encoding/json"
	"fmt"
	"net/http"

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
	}
	msg := extractMessage(resp, "")
	if msg != "" {
		return fmt.Errorf("request failed with message: %s", msg)
	}
	return fmt.Errorf("request failed: %s", resp.Status())
}

func extractMessage(resp *resty.Response, fallback string) string {
	var e ErrorResponse
	if err := json.Unmarshal([]byte(resp.String()), &e); err == nil && e.Message != "" {
		return e.Message
	}
	return fallback
}
