# pkg/mcpserver

MCP server library for multi-tenant deployments. Provides a streamable-HTTP
`http.Handler` that authenticates each request via a caller-supplied
`TenantResolver`, scopes the MCP tool list per tenant, and supports graceful
shutdown that drains in-flight tool calls.

## Quickstart

```go
package main

import (
    "context"
    "log"
    "net/http"

    "github.com/suprsend/cli/pkg/mcpserver"
    "github.com/suprsend/cli/pkg/tenant"
)

func main() {
    creds := tenant.Credentials{
        ServiceToken:      "<your service token>",
        Workspace:         "staging",
        WorkflowsSelector: "all",
        EventsSelector:    "all",
    }
    ctx := tenant.WithCredentials(context.Background(), creds)
    tools, err := mcpserver.BuildTenantTools(ctx)
    if err != nil { log.Fatal(err) }

    resolver := mcpserver.NewFakeResolver()
    resolver.Add("dev-token", &mcpserver.Tenant{Credentials: creds, Tools: tools})

    handler := mcpserver.New(mcpserver.Options{Resolver: resolver})

    // Graceful shutdown — call handler.Shutdown(ctx) before httpServer.Shutdown.
    httpSrv := &http.Server{Addr: ":8080", Handler: handler}
    log.Fatal(httpSrv.ListenAndServe())
}
```

## Implementing `TenantResolver`

```go
type TenantResolver interface {
    Resolve(ctx context.Context, r *http.Request) (*Tenant, error)
}
```

Given an `*http.Request`, return a `*Tenant` (with credentials + the
per-tenant tool set) or an error wrapping `ErrUnauthorized` or `ErrForbidden`
so the outer middleware returns the right HTTP status. Implementations must
be safe for concurrent use.

A multi-tenant server typically:

1. Extracts a bearer token (or other credential) from the request.
2. Looks up the workspace / service token from the auth backend.
3. Builds the per-tenant tool set via `mcpserver.BuildTenantTools(ctx)` and
   optionally appends its own tools.
4. Returns the `*Tenant`.

Cache `(token -> Tenant)` aggressively in your resolver. The library does
not cache; pick a TTL that matches your security/staleness trade-off.
Invalidate on HTTP 401 from the management API (handled reactively by
`MarkSessionDead` — see below).

## Reactive session closure on stale credentials

The `internal/utils` HTTP transport interceptor calls
`mcpserver.MarkSessionDead(ctx)` automatically when the management API
responds 401. The session is closed after the in-flight tool call returns;
the client gets HTTP 404 on its next request and must reconnect (which
re-runs your resolver, which catches the revoked token).

You don't need to do anything to enable this — it's wired by default for any
tool handler that uses `utils.GetSuprSendWorkspaceClient(workspace, ctx)`.

Call `MarkSessionDead(ctx)` manually from a handler when you have
out-of-band evidence the session's downstream credentials have become
invalid.

## Observability hooks

```go
mcpserver.New(mcpserver.Options{
    Resolver: myResolver,
    OnSessionStart: func(ctx context.Context, t *mcpserver.Tenant) context.Context {
        // Stash per-session state (trace span, request ID, tenant logger, etc.)
        // on the returned context. Every per-call handler in the session sees
        // the values via ctx.Value lookups.
        return ctx
    },
    OnSessionEnd: func(ctx context.Context, t *mcpserver.Tenant) {
        // Best-effort cleanup. Fires from a background reconciliation
        // goroutine; may lag the actual session end by up to ~SessionTimeout/2.
    },
    OnToolCall: func(ctx context.Context, name string) (context.Context, func(*mcpsdk.Result, error)) {
        // Before-fn returns the per-call context; after-fn runs with the
        // result + error once the handler returns.
        return ctx, func(_ *mcpsdk.Result, _ error) {}
    },
})
```

## Graceful shutdown

```go
shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
if err := mcpHandler.Shutdown(shutdownCtx); err != nil {
    log.Printf("MCP shutdown: %v", err)
}
if err := httpSrv.Shutdown(shutdownCtx); err != nil {
    log.Printf("HTTP shutdown: %v", err)
}
```

Always call `mcpHandler.Shutdown` BEFORE `httpSrv.Shutdown` so MCP sessions
get a chance to drain before the listener closes.

## Stability

See package-level doc on each `pkg/*` package. The pkg/* tree is stable
under v1 SemVer from day one; breaking changes require a v2 major bump.
CHANGELOG.md at the repo root tracks every release.
