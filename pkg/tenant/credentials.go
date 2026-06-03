// Package tenant carries per-request SuprSend credentials on a context.Context.
// Both the CLI (single tenant at startup) and any multi-tenant HTTP server
// using pkg/mcpserver (one tenant per session) populate this; tool handlers
// and utils.GetSuprSendWorkspaceClient read from it.
package tenant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// Credentials identifies a single SuprSend tenant and selects which dynamic
// tool slugs the session is allowed to register.
//
// The ServiceToken field is sensitive. Stringer and MarshalJSON methods on
// this type REDACT the token so it cannot accidentally leak via fmt.Sprintf
// debug prints, structured-logging field dumps, or JSON encoding. Callers
// that genuinely need the raw token must read the field directly.
type Credentials struct {
	// ServiceToken authenticates against the SuprSend management API.
	// Required. Redacted by String() and MarshalJSON().
	ServiceToken string
	// Workspace is the SuprSend workspace name handlers should act on by
	// default when no workspace is named in the tool arguments.
	Workspace string
	// HubBaseURL / MgmntBaseURL override the default SuprSend endpoints.
	// Empty strings select the defaults (hub.suprsend.com / management-api.suprsend.com).
	HubBaseURL   string
	MgmntBaseURL string
	// WorkflowsSelector / EventsSelector follow the existing CLI selector
	// grammar: "all", "none", "slug1,slug2", or "tag:foo". Empty == "none".
	WorkflowsSelector string
	EventsSelector    string
}

// String returns a representation of Credentials with ServiceToken redacted.
// Implements fmt.Stringer so default %v / %+v printing does not leak tokens.
func (c Credentials) String() string {
	return fmt.Sprintf("tenant.Credentials{ServiceToken:[REDACTED %d chars] Workspace:%q HubBaseURL:%q MgmntBaseURL:%q WorkflowsSelector:%q EventsSelector:%q}",
		len(c.ServiceToken), c.Workspace, c.HubBaseURL, c.MgmntBaseURL, c.WorkflowsSelector, c.EventsSelector)
}

// MarshalJSON returns JSON with ServiceToken redacted. Implements
// json.Marshaler so accidental json.Marshal(creds) does not leak the token.
func (c Credentials) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ServiceToken      string `json:"service_token"`
		Workspace         string `json:"workspace,omitempty"`
		HubBaseURL        string `json:"hub_base_url,omitempty"`
		MgmntBaseURL      string `json:"mgmnt_base_url,omitempty"`
		WorkflowsSelector string `json:"workflows_selector,omitempty"`
		EventsSelector    string `json:"events_selector,omitempty"`
	}{
		ServiceToken:      "[REDACTED]",
		Workspace:         c.Workspace,
		HubBaseURL:        c.HubBaseURL,
		MgmntBaseURL:      c.MgmntBaseURL,
		WorkflowsSelector: c.WorkflowsSelector,
		EventsSelector:    c.EventsSelector,
	})
}

type contextKey struct{}

var key = contextKey{}

// ErrNoCredentials is returned by FromContext when no credentials are set.
var ErrNoCredentials = errors.New("tenant: no credentials on context")

// WithCredentials returns a copy of ctx carrying creds.
func WithCredentials(ctx context.Context, creds Credentials) context.Context {
	return context.WithValue(ctx, key, creds)
}

// FromContext returns the credentials on ctx, or ErrNoCredentials.
func FromContext(ctx context.Context) (Credentials, error) {
	v, ok := ctx.Value(key).(Credentials)
	if !ok {
		return Credentials{}, ErrNoCredentials
	}
	return v, nil
}
