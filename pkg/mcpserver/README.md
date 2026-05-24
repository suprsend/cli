# pkg/mcpserver

OSS hosted-server library for SuprSend MCP. The closed-source deployment binary
imports this package, supplies a `TenantResolver` that calls the bridge API,
and runs `http.ListenAndServe` on the returned handler.

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

    // Graceful shutdown — call mcpHandler.Shutdown(ctx) before httpServer.Shutdown.
    httpSrv := &http.Server{Addr: ":8080", Handler: handler}
    log.Fatal(httpSrv.ListenAndServe())
}
```

## Production: implementing `TenantResolver`

Implement the `TenantResolver` interface against your auth backend. The
contract is one method: given an `*http.Request`, return a `*Tenant` (with
credentials + the per-tenant tool set) or an error wrapping `ErrUnauthorized`
or `ErrForbidden` so the outer middleware returns the right HTTP status.

```go
import "fmt"

func (r *bridgeResolver) Resolve(ctx context.Context, req *http.Request) (*mcpserver.Tenant, error) {
    token := bearer(req)
    if token == "" {
        return nil, fmt.Errorf("missing bearer: %w", mcpserver.ErrUnauthorized)
    }
    workspace, err := r.lookupWorkspace(ctx, token) // bridge API call, cached
    if err != nil {
        return nil, fmt.Errorf("bridge lookup: %w", mcpserver.ErrUnauthorized)
    }

    creds := tenant.Credentials{
        ServiceToken:      token,
        Workspace:         workspace,
        WorkflowsSelector: "all",
        EventsSelector:    "all",
    }
    ctx = tenant.WithCredentials(ctx, creds)
    tools, err := mcpserver.BuildTenantTools(ctx)
    if err != nil { return nil, err }
    // Append closed-source proprietary tools after the OSS static + dynamic set.
    tools = append(tools, myProprietaryTools...)
    return &mcpserver.Tenant{Credentials: creds, Tools: tools}, nil
}
```

Cache `(token -> workspace)` aggressively in your resolver. The OSS library
deliberately owns no cache (Question-5 decision); pick a TTL that matches
your security/staleness trade-off. Invalidate on 401 from the management API
(handled reactively by `MarkSessionDead` — see below).

## Reactive session closure on stale credentials

The `internal/utils/sdk_instance.go` transport interceptor calls
`mcpserver.MarkSessionDead(ctx)` automatically when the management API
responds 401. The session is closed after the in-flight tool call returns;
the client gets HTTP 404 on its next request and must reconnect (which
re-runs your resolver, which catches the revoked token).

You don't need to do anything to enable this — it's wired by default for any
tool handler that uses `utils.GetSuprSendWorkspaceClient(workspace, ctx)`.

## Observability hooks

```go
mcpserver.New(mcpserver.Options{
    Resolver: myResolver,
    OnSessionStart: func(ctx context.Context, t *mcpserver.Tenant) context.Context {
        span := tracer.StartSpan("mcp.session", trace.WithAttributes(
            attribute.String("tenant.workspace", t.Credentials.Workspace),
        ))
        return trace.ContextWithSpan(ctx, span)
    },
    OnSessionEnd: func(ctx context.Context, _ *mcpserver.Tenant) {
        if span := trace.SpanFromContext(ctx); span != nil {
            span.End()
        }
    },
    OnToolCall: func(ctx context.Context, name string) (context.Context, func(*mcpsdk.Result, error)) {
        span := tracer.StartSpan("mcp.tool."+name)
        return trace.ContextWithSpan(ctx, span), func(r *mcpsdk.Result, err error) {
            if err != nil || (r != nil && r.IsError) {
                span.RecordError(err)
            }
            span.End()
        }
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

## Local development against an unreleased branch

The closed-source repo is a separate Go module pinning a tag of this OSS
repo (Question-b decision). To test against an unreleased branch:

```bash
# in the closed-source repo
go mod edit -replace github.com/suprsend/cli=../cli
go mod tidy
```

Remove the `replace` directive before committing.

## Stability

See package-level doc on each `pkg/*` package. The pkg/* tree is stable
under v1 SemVer from day one (Question-c decision); breaking changes
require a v2 major bump. CHANGELOG.md at the repo root tracks every release.
