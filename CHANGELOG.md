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
- **`pkg/mcpserver`** — OSS hosted-server library: `New(Options) *Handler`,
  `TenantResolver` interface, `Tenant`, `BuildTenantTools`, `MarkSessionDead`,
  `FakeResolver`. Sentinel errors `ErrUnauthorized`, `ErrForbidden` for outer
  auth middleware HTTP status mapping. Observability hooks `OnSessionStart`,
  `OnSessionEnd`, `OnToolCall`. Graceful shutdown via `(*Handler).Shutdown(ctx)`.
  Stateful streamable HTTP transport with hosted capability profile
  (recovery + tools(listChanged) + resources(listChanged) + logging).
  Outer auth middleware enforces session-tenant binding to mitigate
  Mcp-Session-Id theft. Background reconciliation goroutine bounds memory.

### Changed

- **`internal/utils.GetSuprSendWorkspaceClient`** — now accepts a variadic
  `context.Context` so handlers can pass the per-tenant context. The legacy
  1-arg form remains source-compatible.
- **`mgmnt`** — refactored to accept an injectable `http.RoundTripper` via
  `NewClientWithUrlsAndTransport`, applied through `(c *SS_MgmntClient).restyClient()`
  to all ~40 management API call sites. Added 10s bridge-call timeout
  (was previously unbounded). All eight `internal/tools/*.go` files ported
  from the legacy `mark3labs/mcp-go` API to `pkg/mcpsdk.Tool` abstraction,
  with per-handler `utils.IsAuthError → mcpserver.MarkSessionDead` discipline
  on every SuprSend API error path.
- **CLI's `start-mcp-server`** — migrated from `github.com/mark3labs/mcp-go`
  to `github.com/modelcontextprotocol/go-sdk`. No user-visible behavior change
  expected. CLI capability profile: recovery + tools (no listChanged).

### Removed

- **`github.com/mark3labs/mcp-go`** dependency dropped after all tool files
  ported to the official SDK.

## [1.0.0] and earlier

(Existing CLI release history — backfill from git tags as needed.)
