# Closed-Source Hosted MCP Server — Reference Skeleton

This describes the expected shape of the private deployment binary that wraps
`github.com/suprsend/cli/pkg/mcpserver`. The skeleton below should live in the
private repo, not in the OSS CLI repo.

## main.go

```go
package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "strings"
    "syscall"
    "time"

    "github.com/suprsend/cli/pkg/mcpserver"
)

const (
    maxRequestBodyBytes = 8 << 20 // 8 MB — tool args are usually small; bigger is suspicious
    readHeaderTimeout   = 5 * time.Second
    readTimeout         = 30 * time.Second
    writeTimeout        = 60 * time.Second  // tool calls may stream; keep generous but bounded
    idleTimeout         = 90 * time.Second
    maxHeaderBytes      = 1 << 16 // 64 KB
    shutdownGrace       = 30 * time.Second
)

func main() {
    resolver := newBridgeResolver(bridgeResolverConfig{
        HubBaseURL:   envOr("SUPRSEND_HUB_BASE_URL", "https://hub.suprsend.com"),
        MgmntBaseURL: envOr("SUPRSEND_MGMNT_BASE_URL", "https://management-api.suprsend.com/"),
        TTL:          5 * time.Minute,
    })

    mcpHandler := mcpserver.New(mcpserver.Options{
        Resolver:       resolver,
        OnSessionStart: traceSessionStart,
        OnSessionEnd:   traceSessionEnd,
        OnToolCall:     traceToolCall,
    })

    // Outer middleware stack:
    //   1. observability (request logs, metrics) — outermost so it sees everything
    //   2. cross-origin protection — denies untrusted origins (CSRF defense)
    //   3. body-size limit — DoS protection
    //   4. mcpHandler (which does auth + MCP)
    cors := http.NewCrossOriginProtection() // Go 1.25+; DEFAULT REJECTS ALL cross-origin POSTs
    // Configure trusted origins from ALLOWED_ORIGINS env var (comma-separated).
    // For browser-based MCP clients (claude.ai, MCP Apps hosts), the origin
    // hosting the client must appear here. Empty list = same-origin only.
    for _, origin := range strings.Split(envOr("ALLOWED_ORIGINS", ""), ",") {
        origin = strings.TrimSpace(origin)
        if origin == "" {
            continue
        }
        if err := cors.AddTrustedOrigin(origin); err != nil {
            log.Fatalf("invalid ALLOWED_ORIGINS entry %q: %v", origin, err)
        }
    }
    var handler http.Handler = mcpHandler
    handler = http.MaxBytesHandler(handler, maxRequestBodyBytes)
    handler = cors.Handler(handler)
    handler = withObservability(handler)

    addr := envOr("LISTEN_ADDR", ":8080")
    httpSrv := &http.Server{
        Addr:              addr,
        Handler:           handler,
        ReadHeaderTimeout: readHeaderTimeout,
        ReadTimeout:       readTimeout,
        WriteTimeout:      writeTimeout,
        IdleTimeout:       idleTimeout,
        MaxHeaderBytes:    maxHeaderBytes,
    }

    go func() {
        log.Printf("hosted MCP listening on %s", addr)
        if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("http: %v", err)
        }
    }()

    // Graceful shutdown — drain MCP sessions BEFORE closing the listener.
    sig := make(chan os.Signal, 1)
    signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
    <-sig

    shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownGrace)
    defer cancel()
    if err := mcpHandler.Shutdown(shutdownCtx); err != nil {
        log.Printf("MCP shutdown: %v", err)
    }
    if err := httpSrv.Shutdown(shutdownCtx); err != nil {
        log.Printf("HTTP shutdown: %v", err)
    }
}

func envOr(key, def string) string {
    if v := os.Getenv(key); v != "" { return v }
    return def
}
```

### Security middleware stack — what each layer does

| Layer                     | Reason                                                                                          |
|---------------------------|-------------------------------------------------------------------------------------------------|
| `MaxBytesHandler` (8 MB)  | DoS protection. SDK does `io.ReadAll(req.Body)` in some paths — without a cap, a hostile POST can OOM the pod. |
| `CrossOriginProtection`   | MCP clients may be browser-based (Claude.ai, MCP Apps hosts). Blocks unauthorized cross-origin POSTs. **Default rejects ALL cross-origin** — configure `ALLOWED_ORIGINS` env var (comma-separated) with the origins of your supported browser MCP clients. Empty = same-origin only (CLI/server-to-server clients only). |
| `http.Server` timeouts    | Slowloris / connection exhaustion mitigation. Defaults to no timeouts in Go — explicit values mandatory in production. |
| `MaxHeaderBytes` (64 KB)  | Header-bomb protection. Default 1 MB is too permissive for an API endpoint. |
| `shutdownGrace` (30 s)    | Long enough to drain typical tool calls; short enough that k8s sigkill doesn't cut it off. |

## bridge_resolver.go

```go
type bridgeResolver struct {
    cfg   bridgeResolverConfig
    cache *ttlcache.Cache[string, cachedTenant] // (bearer-token -> tenant), TTL 5 min
}

func (b *bridgeResolver) Resolve(ctx context.Context, r *http.Request) (*mcpserver.Tenant, error) {
    token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
    if token == "" {
        return nil, fmt.Errorf("missing bearer token: %w", mcpserver.ErrUnauthorized)
    }
    if cached, ok := b.cache.Get(token); ok {
        return cached.tenant, nil
    }
    workspace, err := b.lookupWorkspaceViaBridge(ctx, token)
    if err != nil {
        // Wrap with sentinel so the OSS auth middleware returns HTTP 401, not 500.
        return nil, fmt.Errorf("bridge lookup: %w", mcpserver.ErrUnauthorized)
    }
    creds := tenant.Credentials{
        ServiceToken: token, Workspace: workspace,
        HubBaseURL: b.cfg.HubBaseURL, MgmntBaseURL: b.cfg.MgmntBaseURL,
        WorkflowsSelector: "all", EventsSelector: "all",
    }
    ctx = tenant.WithCredentials(ctx, creds)
    tools, err := mcpserver.BuildTenantTools(ctx)
    if err != nil { return nil, err }
    // Append proprietary closed-source tools after the OSS set.
    tools = append(tools, myProprietaryTools...)
    t := &mcpserver.Tenant{Credentials: creds, Tools: tools}
    b.cache.Set(token, cachedTenant{tenant: t}, b.cfg.TTL)
    return t, nil
}
```

## Wiring observability + graceful shutdown

```go
handler := mcpserver.New(mcpserver.Options{
    Resolver:       resolver,
    OnSessionStart: traceSessionStart, // OTel span
    OnSessionEnd:   traceSessionEnd,
    OnToolCall:     traceToolCall,     // per-call span + metric
})

httpSrv := &http.Server{
    Addr:    envOr("LISTEN_ADDR", ":8080"),
    Handler: withRateLimit(withRequestLog(handler)),
}

go func() {
    if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        log.Fatalf("http: %v", err)
    }
}()

sig := make(chan os.Signal, 1)
signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
<-sig

shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
if err := handler.Shutdown(shutdownCtx); err != nil {
    log.Printf("MCP shutdown: %v", err)
}
if err := httpSrv.Shutdown(shutdownCtx); err != nil {
    log.Printf("HTTP shutdown: %v", err)
}
```

## Operational notes

- **Cache TTL**: 5 min default, configurable. The OSS library does not cache —
  this resolver is the only cache (Question-5 decision).
- **Reactive session closure on stale credentials**: handled automatically by
  the OSS library when a tool's downstream API call returns 401. The next
  request from that client gets HTTP 404 and reconnects, which re-runs
  Resolve and catches the revoked token.
- **Rate limiting + structured request logs**: wrap the handler in your own
  HTTP middleware (`withRateLimit`, `withRequestLog`). The OSS library is
  intentionally observability-agnostic — use the three hooks for MCP-internal
  observability (sessions, tool calls), HTTP middleware for transport-level.
- **MCP-Session-Id header**: the SDK handles session reuse automatically.
  The OSS auth middleware (running per request) re-runs Resolve on every
  request — your cache absorbs the cost.
- **Bearer-token auth**: fine for v1. OAuth 2.1 wrapper is a follow-up if/when
  the hosted server gets listed on the public MCP Registry.
- **`go mod replace` for local dev**: when testing against an unreleased OSS
  branch, `go mod edit -replace github.com/suprsend/cli=../cli && go mod tidy`.
  Remove the replace before committing.
