# Changelog

All notable changes to `github.com/suprsend/cli` are documented here. Format
follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/); this repo
adheres to [Semantic Versioning](https://semver.org/).

The `pkg/*` tree (pkg/mcpsdk, pkg/tenant, pkg/mcpserver) is part of the v1
stability commitment from initial release. Breaking changes to anything under
`pkg/*` require a v2 major bump.

## [Unreleased]

### Added

- **`pkg/mcpsdk`** — runtime-agnostic MCP tool representation: `Tool`, `Args`,
  `Result`, `Annotations`, `ToolHandler`.
- **`pkg/tenant`** — per-request SuprSend credentials on `context.Context`:
  `Credentials`, `WithCredentials`, `FromContext`, `ErrNoCredentials`.
  Credentials' service token is redacted by overridden `String()` and
  `MarshalJSON()` to prevent accidental leaks via debug prints / structured
  logging.
- **`pkg/mcpserver`** — MCP server library for multi-tenant deployments:
  `New(Options) *Handler`, `TenantResolver` interface, `Tenant`,
  `BuildTenantTools`, `MarkSessionDead`, `FakeResolver`. Sentinel errors
  `ErrUnauthorized`, `ErrForbidden` for outer auth middleware HTTP status
  mapping. Observability hooks `OnSessionStart`, `OnSessionEnd`, `OnToolCall`.
  Graceful shutdown via `(*Handler).Shutdown(ctx)`. Stateful streamable HTTP
  transport with a default capability profile (recovery + tools(listChanged)
  + resources(listChanged) + logging). Outer auth middleware enforces
  session-tenant binding to prevent stolen-session-ID reuse across tenants.
  Background reconciliation goroutine bounds memory.
- **`pkg/mcpserver.Options.UnauthorizedChallenge`** — optional
  `func(missing bool) string` to override the `WWW-Authenticate` header sent on
  401 responses. `missing` is true when no credential was resolved and false
  when a credential was presented but rejected, so embedders can emit
  `error="invalid_token"` and a `resource_metadata` pointer per RFC 9728 for
  MCP OAuth discovery. Returning `""` suppresses the header. Defaults to
  `Bearer realm="suprsend"` when nil (unchanged behavior).

### Changed

- **`internal/utils.GetSuprSendWorkspaceClient`** — now accepts a variadic
  `context.Context` so handlers can pass the per-tenant context. The legacy
  1-arg form remains source-compatible.
- **`mgmnt`** — refactored to accept an injectable `http.RoundTripper` via
  `NewClientWithUrlsAndTransport`, applied through `(c *SS_MgmntClient).restyClient()`
  to all ~40 management API call sites. Added 10s timeout on the workspace
  key/secret lookup (was previously unbounded). All eight
  `internal/tools/*.go` files ported from the legacy `mark3labs/mcp-go` API
  to `pkg/mcpsdk.Tool` abstraction, with per-handler
  `utils.IsAuthError → mcpserver.MarkSessionDead` discipline on every
  SuprSend API error path.
- **CLI's `start-mcp-server`** — migrated from `github.com/mark3labs/mcp-go`
  to `github.com/modelcontextprotocol/go-sdk`. No user-visible behavior change
  expected. CLI capability profile: recovery + tools (no listChanged).
- **`github.com/suprsend/suprsend-go`** bumped to v0.10.1, which propagates the
  caller's `context.Context` into its HTTP requests (`NewRequestWithContext`).
  Event-trigger and workflow-trigger handlers now call the `WithContext`
  method variants.
- **`mgmnt`** management-API client now threads `context.Context` through all
  44 HTTP methods (`.SetContext(ctx)` on every resty request). Combined with the
  suprsend-go v0.10.1 bump, **every** SuprSend/mgmnt client now propagates the
  request context, so the `authExpiryTransport` 401 interceptor sees the
  per-session dead-flag on `req.Context()` for all calls. This makes the
  transport interceptor the single reactive-session-close mechanism and the
  per-handler `utils.IsAuthError → markSessionDead` discipline (≈47 call sites
  across the tool handlers) was **removed** as redundant; `utils.IsAuthError`
  is gone. Also enables cancellation/deadline propagation for mgmnt calls.

### Fixed

- **`start-mcp-server --tools` exact-name selectors** — the official-SDK
  migration renamed every tool's protocol name (`users.get` →
  `get_suprsend_user`, etc.), which silently broke `--tools=users.get`-style
  selectors (they matched nothing and registered zero tools). The selector
  grammar now resolves the stable legacy names (advertised in the command help)
  to the new protocol names via a decoupled `Selector` field, so existing
  `--tools=users.get,tenants.get_all` invocations work again. Category
  wildcards (`--tools=users.*`) were unaffected.
- **`pkg/mcpserver.FakeResolver`** now returns `(nil, nil)` for a request with
  no `Authorization` header (previously `ErrUnauthorized`). A missing header
  carries no credential, so the auth middleware now reports `missing=true`
  (RFC 9728 discovery) instead of `missing=false` ("credential rejected").
  A present-but-unknown token still returns `ErrUnauthorized`.

### Removed

- **`github.com/mark3labs/mcp-go`** dependency dropped after all tool files
  ported to the official SDK.

## [1.0.0] and earlier

(Existing CLI release history — backfill from git tags as needed.)
