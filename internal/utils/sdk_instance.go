package utils

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/mgmnt"
	"github.com/suprsend/cli/pkg/tenant"
	suprsend "github.com/suprsend/suprsend-go"
)

// SDKInstance is the singleton mgmnt client populated at CLI startup.
// Multi-tenant HTTP servers using pkg/mcpserver do NOT use this — they
// construct per-tenant clients via MgmntClientFor below.
var SDKInstance *mgmnt.SS_MgmntClient

var (
	mgmntCacheMu sync.RWMutex
	mgmntCache   = map[mgmntCacheKey]*mgmnt.SS_MgmntClient{}
)

type mgmntCacheKey struct {
	ServiceToken string
	HubBaseURL   string
	MgmntBaseURL string
}

// InitSDKWithUrls initializes the singleton mgmnt client used by the CLI.
func InitSDKWithUrls(serviceToken string, baseUrl string, mgmntUrl string, debug bool) {
	if SDKInstance != nil {
		log.Error("SDK already initialized")
		return
	}
	SDKInstance = mgmnt.NewClientWithUrls(serviceToken, baseUrl, mgmntUrl, debug)
}

// GetSuprSendMgmntClient returns the singleton mgmnt SDK initialized by
// InitSDKWithUrls. Used by code paths that have not been migrated to the
// context-aware MgmntClientFor.
func GetSuprSendMgmntClient() *mgmnt.SS_MgmntClient { return SDKInstance }

// MgmntClientFor returns a mgmnt client for the credentials on ctx. Clients
// are cached by (service token, hub URL, mgmnt URL). When ctx has no tenant
// credentials (e.g. the CLI before InitSDKWithUrls completes), falls back to
// the singleton SDKInstance.
//
// The returned client's underlying HTTP transport is wrapped with
// authExpiryTransport, which calls mcpserver.MarkSessionDead on every HTTP
// 401 response from any SuprSend API call. In a multi-tenant MCP server
// using pkg/mcpserver, the SDK closes the session after the in-flight tool
// call returns; the client must reconnect (and re-authenticate). In CLI
// mode the MarkSessionDead call is a no-op because the per-request context
// has no dead-session flag installed.
func MgmntClientFor(ctx context.Context) *mgmnt.SS_MgmntClient {
	creds, err := tenant.FromContext(ctx)
	if err != nil {
		return SDKInstance
	}
	k := mgmntCacheKey{ServiceToken: creds.ServiceToken, HubBaseURL: creds.HubBaseURL, MgmntBaseURL: creds.MgmntBaseURL}
	mgmntCacheMu.RLock()
	if c, ok := mgmntCache[k]; ok {
		mgmntCacheMu.RUnlock()
		return c
	}
	mgmntCacheMu.RUnlock()

	mgmntCacheMu.Lock()
	defer mgmntCacheMu.Unlock()
	if c, ok := mgmntCache[k]; ok {
		return c
	}
	rt := &authExpiryTransport{Base: http.DefaultTransport}
	c := mgmnt.NewClientWithUrlsAndTransport(creds.ServiceToken, creds.HubBaseURL, creds.MgmntBaseURL, rt, false)
	mgmntCache[k] = c
	return c
}

// GetSuprSendWorkspaceClient returns the suprsend SDK client for the named
// workspace. If ctx carries tenant credentials, the client is scoped to that
// tenant; otherwise it falls back to the process singleton.
//
// The variadic ctx is for source-compatibility with the legacy 1-arg form
// during the migration. New callers should pass exactly one context.
func GetSuprSendWorkspaceClient(workspace string, ctx ...context.Context) (*suprsend.Client, error) {
	c := SDKInstance
	if len(ctx) > 0 {
		c = MgmntClientFor(ctx[0])
	}
	if c == nil {
		return nil, fmt.Errorf("no SuprSend client available (no tenant credentials on context and no initialized SDK)")
	}
	// Thread the request context into the per-tenant workspace lookup so the
	// authExpiryTransport's 401 handling and cancellation propagate. Legacy
	// 1-arg callers fall back to the background-ctx variant.
	if len(ctx) > 0 {
		return c.GetWorkspaceClientCtx(ctx[0], workspace)
	}
	return c.GetWorkspaceClient(workspace)
}

// authExpiryTransport wraps an http.RoundTripper. When the wrapped response
// is HTTP 401, it calls mcpserver.MarkSessionDead with the request's ctx —
// which signals the per-session dead-flag installed by pkg/mcpserver's
// per-session middleware. In CLI contexts (no dead-flag), the call is a
// harmless no-op.
//
// Note: a 401 from a TRANSIENT auth-backend outage will also trigger close,
// which is a false positive. The cost is small — the client reconnects and
// re-auths. We accept this trade-off because the cost of NOT closing on a
// true revocation is unbounded continued access with stale credentials.
type authExpiryTransport struct {
	Base http.RoundTripper
}

func (t *authExpiryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	resp, err := base.RoundTrip(req)
	if err == nil && resp != nil && resp.StatusCode == http.StatusUnauthorized {
		if MarkSessionDead != nil {
			MarkSessionDead(req.Context())
		}
	}
	return resp, err
}

// MarkSessionDead is assigned by pkg/mcpserver.init() to its MarkSessionDead
// function, breaking the would-be import cycle (utils → mcpserver → tools →
// utils). Nil in CLI-only builds; calls on it are guarded.
//
// Any binary that imports pkg/mcpserver triggers the init() assignment so
// the transport interceptor's call becomes effective.
var MarkSessionDead func(ctx context.Context)

// IsAuthError reports whether err represents an authentication failure from a
// SuprSend API call. Used by tool handlers in their error paths to call
// mcpserver.MarkSessionDead(ctx) when the underlying API returned 401.
//
// Two mechanisms close a session on a revoked token; this is the backstop:
//
//   - Primary (transport interceptor): authExpiryTransport calls
//     MarkSessionDead(req.Context()) on any 401. As of suprsend-go v0.10.1 the
//     SDK propagates the handler's context into its HTTP requests
//     (prepareHttpRequest now uses http.NewRequestWithContext), so this fires
//     for every suprsend-go call that threads ctx — i.e. all of them. The same
//     holds for the mgmnt workspace key/secret lookup, which also builds its
//     request with NewRequestWithContext.
//   - Backstop (this function): the resty-based mgmnt management API client
//     (GetSchema, GetWorkflows, etc.) does NOT yet propagate ctx, so a 401 from
//     those calls won't reach the transport interceptor. Per-handler IsAuthError
//     covers that path and provides defense-in-depth for the rest.
//     MarkSessionDead is idempotent, so the double-signal is harmless.
//
// Implementation, in priority order:
//
//  1. errors.As against *suprsend.Error (the typed error returned by
//     suprsend-go's Users/Events/Workflow APIs). Its Code field carries the
//     HTTP status; Code == 401 is an auth failure.
//  2. errors.As against *clierr.CLIError (the typed error mgmnt/errors.go
//     returns). A 401 is classified there as clierr.CodeAuthInvalidToken.
//  3. Defensive heuristic fallback: if no typed signal is found, match the
//     error message (case-insensitive) for "401" or "unauthorized". This is
//     a best-effort guard against errors that lost their type through string
//     wrapping; it can theoretically false-positive on an unrelated message
//     mentioning "401", but the cost of a false positive (one needless
//     session teardown + reconnect) is bounded, while missing a true
//     revocation is not.
func IsAuthError(err error) bool {
	if err == nil {
		return false
	}

	// 1. suprsend-go typed error.
	var ssErr *suprsend.Error
	if errors.As(err, &ssErr) && ssErr.Code == http.StatusUnauthorized {
		return true
	}

	// 2. mgmnt typed error.
	var ce *clierr.CLIError
	if errors.As(err, &ce) && ce.Code == clierr.CodeAuthInvalidToken {
		return true
	}

	// 3. Heuristic fallback for untyped/string-wrapped errors.
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "401") || strings.Contains(msg, "unauthorized") {
		return true
	}

	return false
}
