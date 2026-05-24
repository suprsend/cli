package mcpserver

import (
	"context"
	"errors"
	"net/http"
)

// newAuthMiddleware wraps next in an http.Handler that runs Resolver.Resolve
// on every request, maps sentinel errors to HTTP status codes, enforces
// session-tenant binding to prevent stolen-session-ID reuse, and stashes the
// resolved *Tenant on the request context for getServer to pick up.
//
// Auth runs per-request (not per-session) so revoked tokens stop working
// immediately rather than at session expiry. The session-tenant binding
// prevents Mcp-Session-Id theft: an attacker presenting a stolen session ID
// with a different valid bearer token gets HTTP 403 because the bearer's
// resolved tenant identifier does not match the session's original.
func newAuthMiddleware(resolver TenantResolver, h *Handler, next http.Handler) http.Handler {
	const sessionIDHeader = "Mcp-Session-Id"

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.closing.Load() {
			// Don't accept new requests after Shutdown starts. Tell the
			// load balancer to retry later (no Retry-After header for now;
			// embedders can wrap us if they want finer control).
			http.Error(w, "Service Unavailable: shutting down", http.StatusServiceUnavailable)
			return
		}

		tenantRec, err := resolver.Resolve(r.Context(), r)
		if err != nil {
			switch {
			case errors.Is(err, ErrUnauthorized):
				w.Header().Set("WWW-Authenticate", `Bearer realm="suprsend"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
			case errors.Is(err, ErrForbidden):
				http.Error(w, "Forbidden", http.StatusForbidden)
			default:
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}
		if tenantRec == nil {
			// Resolver returned (nil, nil) — treat as unauthorized.
			w.Header().Set("WWW-Authenticate", `Bearer realm="suprsend"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Session-tenant binding. The binding is registered by the per-call
		// GetSessionID hook in getServer (server.go) at the moment the SDK
		// assigns a session ID — that's the only point at which we know the
		// ID. Here we read the binding on subsequent requests: if a session
		// ID we know maps to a different tenant than the bearer just
		// resolved to, the request is rejected with HTTP 403 (a stolen
		// session ID cannot be reused across tenants). Unknown session IDs
		// skip the check (first request of a session has no Mcp-Session-Id
		// header; second request and onward will have it).
		if sessID := r.Header.Get(sessionIDHeader); sessID != "" {
			if originalIdent, ok := h.sessionTenants.Load(sessID); ok {
				if originalIdent.(string) != tenantIdentifier(tenantRec.Credentials) {
					// Log the security event so embedders can alert. Log a
					// PREFIX of the session ID only (full ID is sensitive).
					if logger := h.serverLogger(); logger != nil {
						prefixLen := 8
						if len(sessID) < prefixLen {
							prefixLen = len(sessID)
						}
						logger.Warn("mcpserver: session-hijack attempt rejected",
							"session_id_prefix", sessID[:prefixLen]+"...",
							"client_addr", r.RemoteAddr,
							"user_agent", r.Header.Get("User-Agent"),
						)
					}
					// Respond identically to "unknown session" — don't leak
					// whether the session ID was valid for a different tenant.
					http.Error(w, "Forbidden", http.StatusForbidden)
					return
				}
			}
		}

		ctx := context.WithValue(r.Context(), resolvedTenantKey{}, tenantRec)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
