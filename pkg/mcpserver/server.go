// Package mcpserver is an MCP server library supporting multi-tenant
// deployments. Embedders supply a TenantResolver that authenticates each
// incoming HTTP request and resolves it to a per-tenant credential set + tool
// list; the library serves MCP over streamable HTTP with per-tenant tool
// scoping. Call Handler.Shutdown(ctx) during graceful shutdown to drain
// in-flight tool calls before the listener closes.
//
// Pair with FakeResolver to spin up a single-tenant server for local
// development, or with a custom TenantResolver to integrate with any auth
// backend.
package mcpserver

import (
	"context"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/suprsend/cli/internal/tools"
	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/cli/pkg/mcpsdk"
	"github.com/suprsend/cli/pkg/tenant"
)

func init() {
	// Break the would-be import cycle: utils' authExpiryTransport calls
	// MarkSessionDead but utils cannot import this package (or the DeadOption
	// type). We register a thin wrapper that drops the variadic — utils only
	// needs the "kill this session" signal, never options. CLI-only builds
	// that don't link pkg/mcpserver leave utils.MarkSessionDead nil → the
	// transport interceptor's call is a no-op.
	utils.MarkSessionDead = func(ctx context.Context) { MarkSessionDead(ctx) }
}

// Sentinel errors. Resolvers wrap these (with fmt.Errorf("...: %w", ErrXxx))
// to control the HTTP status returned by the outer auth middleware. Any
// resolver error that does not match either sentinel becomes HTTP 500.
var (
	// ErrUnauthorized — auth credentials missing or invalid. Outer middleware
	// returns HTTP 401 with WWW-Authenticate: Bearer realm="suprsend".
	ErrUnauthorized = errors.New("mcpserver: unauthorized")
	// ErrForbidden — auth credentials valid but caller lacks permission.
	// Outer middleware returns HTTP 403.
	ErrForbidden = errors.New("mcpserver: forbidden")
)

// Tenant is what a TenantResolver returns. Credentials become the per-handler
// context payload; Tools is the runtime-agnostic tool set this session should
// see. The canonical way for resolvers to populate Tools — including the
// per-tenant dynamic workflow/event-trigger tools — is to call BuildTenantTools
// with a context carrying the credentials:
//
//	ctx = tenant.WithCredentials(ctx, creds)
//	tools, err := mcpserver.BuildTenantTools(ctx)
//	if err != nil { return nil, err }
//	return &mcpserver.Tenant{Credentials: creds, Tools: tools}, nil
//
// Out-of-tree resolvers MUST use BuildTenantTools because they cannot import
// internal/tools directly.
type Tenant struct {
	Credentials tenant.Credentials
	Tools       []*mcpsdk.Tool
}

// BuildTenantTools returns the per-tenant tool set for the credentials on ctx:
// the workspace-agnostic static tools plus the tenant's dynamic
// workflow/event-trigger tools, scoped by the WorkflowsSelector and
// EventsSelector on the credentials.
//
// Selector grammar matches the CLI's --workflows / --events flags: "all",
// "none", a comma-separated slug list, or "tag:<tag>" entries (mixed forms
// also work). An empty string is treated as "none".
//
// This helper exists so out-of-tree consumers never need to import
// internal/tools directly. It is a thin convenience over
// tools.GetAllTools / tools.RegisterDynamicWorkflowToolsFor / tools.RegisterDynamicEventsToolsFor.
func BuildTenantTools(ctx context.Context) ([]*mcpsdk.Tool, error) {
	creds, err := tenant.FromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("mcpserver: BuildTenantTools: %w", err)
	}
	static := tools.GetAllTools()
	wf, err := tools.RegisterDynamicWorkflowToolsFor(ctx, creds.Workspace, creds.WorkflowsSelector)
	if err != nil {
		return nil, fmt.Errorf("mcpserver: workflow tool registration: %w", err)
	}
	ev, err := tools.RegisterDynamicEventsToolsFor(ctx, creds.Workspace, creds.EventsSelector)
	if err != nil {
		return nil, fmt.Errorf("mcpserver: event tool registration: %w", err)
	}
	out := make([]*mcpsdk.Tool, 0, len(static)+len(wf)+len(ev))
	for _, envelope := range static {
		out = append(out, envelope.Tool)
	}
	for _, envelope := range wf {
		out = append(out, envelope.Tool)
	}
	for _, envelope := range ev {
		out = append(out, envelope.Tool)
	}
	return out, nil
}

// TenantResolver authenticates an incoming HTTP request and resolves it to a
// Tenant. Wrap returned errors with ErrUnauthorized or ErrForbidden to
// control the HTTP response status.
//
// Implementations must be safe for concurrent use. Caching (token -> tenant)
// is the resolver's responsibility — pkg/mcpserver does not cache.
type TenantResolver interface {
	Resolve(ctx context.Context, r *http.Request) (*Tenant, error)
}

// Options configures New. Required fields: Resolver.
type Options struct {
	// Resolver authenticates each incoming HTTP request. REQUIRED.
	Resolver TenantResolver

	// Implementation identifies this server in the MCP initialize response.
	// Defaults to {Name: "suprsend", Version: "dev"} when nil.
	Implementation *mcp.Implementation

	// ServerOptions is the BASE *mcp.ServerOptions. The library's default
	// capability profile (Capabilities: logging) REPLACES base.Capabilities
	// (see mergeServerOptions); all other base fields are preserved. The
	// official SDK additionally infers tools.listChanged from the AddTool
	// calls the library makes. Resources and prompts are NOT advertised.
	// Recovery is installed as outermost middleware, not via Capabilities.
	// Pass nil for "use only the default profile".
	ServerOptions *mcp.ServerOptions

	// HTTPOptions is passed through to mcp.NewStreamableHTTPHandler.
	// Pass nil for stateful default behavior.
	HTTPOptions *mcp.StreamableHTTPOptions

	// OnSessionStart fires once per new MCP session, AFTER successful auth.
	// Returns a per-session context that handlers see; use this to stash
	// session-scoped state (request IDs, span contexts, etc.). May be nil.
	OnSessionStart func(ctx context.Context, t *Tenant) context.Context

	// OnSessionEnd fires when the session is detected closed. May be nil.
	//
	// Timing caveat: the SDK exposes no synchronous session-end hook
	// (ServerSessionOptions.onClose is unexported, server.go:1009). This
	// hook fires from the background reconciliation goroutine, which runs
	// every SessionTimeout/2 by default. End detection is therefore
	// best-effort and may lag the actual session end by up to ~SessionTimeout/2.
	// Do not rely on this for synchronous cleanup of per-session resources;
	// use it for metrics/observability where eventual notification is enough.
	OnSessionEnd func(ctx context.Context, t *Tenant)

	// OnToolCall fires before each tool invocation. Use for metering/tracing.
	// The returned context is passed to the handler; the returned func runs
	// after the handler with the result + error. Either return value may be
	// nil/no-op. May be nil.
	OnToolCall func(ctx context.Context, toolName string) (context.Context, func(result *mcpsdk.Result, err error))

	// UnauthorizedChallenge returns the WWW-Authenticate header value sent on
	// 401 responses. `missing` is true when no credential was presented (the
	// resolver returned no tenant) and false when a credential was present but
	// rejected (the resolver returned ErrUnauthorized), so embedders can add
	// error="invalid_token" per RFC 9728. Embedders fronting the server with an
	// OAuth 2.1 authorization server use this to point clients at their
	// protected-resource metadata, e.g.:
	//
	//	Bearer resource_metadata="https://mcp.example.com/.well-known/oauth-protected-resource", error="invalid_token"
	//
	// Returning "" suppresses the header. Defaults to Bearer realm="suprsend"
	// when nil.
	UnauthorizedChallenge func(missing bool) string
}

// Handler is the http.Handler returned by New. Embeds http.Handler so callers
// can pass it directly to http.ListenAndServe. The added Shutdown method
// drains MCP sessions cleanly during graceful shutdown.
type Handler struct {
	http.Handler // the StreamableHTTPHandler wrapped in the auth middleware
	opts         Options

	servers        sync.Map // *mcp.Server -> serverEntry
	sessionTenants sync.Map // sessionID (string) -> tenantIdentifier (string)
	sessionInfo    sync.Map // sessionID (string) -> *endHookCtx

	// reconcileGrace is the minimum age a sessionless server must reach before
	// reconciliation is allowed to evict it (Fix #9). Set in New from the
	// reconcile interval.
	reconcileGrace time.Duration

	// closing is an advisory fast-path flag for the auth middleware's 503
	// check and getServer's early bail. The authoritative shutdown
	// coordination — gating in-flight tool calls against Shutdown's drain —
	// lives in inflight, whose mutex makes add() and beginShutdown() mutually
	// exclusive (Fix #8: no Add-after-Wait race).
	closing         atomic.Bool
	inflight        inflight
	shutdownOnce    sync.Once
	reconcileStop   chan struct{}
	reconcileDoneCh chan struct{}
}

// inflight is a mutex-coordinated in-flight tool-call tracker. It replaces a
// bare sync.WaitGroup so that incrementing the counter (add) and the
// closing-check that gates Shutdown's drain (beginShutdown) are mutually
// exclusive. With a raw WaitGroup, a call that passed the closing fast-path
// could reach Add(1) AFTER Shutdown's Wait() observed zero and returned,
// violating the WaitGroup contract. Here add() and beginShutdown() take the
// same mutex, so once shutdown begins no new call is ever counted.
type inflight struct {
	mu      sync.Mutex
	n       int
	closing bool
	drainCh chan struct{} // non-nil only while closing && n>0; closed when drained
}

// add registers a starting tool call. Returns false if shutdown has already
// begun, in which case the call is NOT counted and Shutdown will not wait for
// it (the session is closed in Shutdown's server-walk regardless).
func (i *inflight) add() bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.closing {
		return false
	}
	i.n++
	return true
}

// done marks a previously-add()'d call complete.
func (i *inflight) done() {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.n--
	if i.closing && i.n == 0 && i.drainCh != nil {
		close(i.drainCh)
		i.drainCh = nil
	}
}

// beginShutdown flips the tracker into closing mode and returns a channel that
// is closed once all in-flight calls have drained (immediately if none).
func (i *inflight) beginShutdown() <-chan struct{} {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.closing = true
	ch := make(chan struct{})
	if i.n == 0 {
		close(ch)
	} else {
		i.drainCh = ch
	}
	return ch
}

// endHookCtx carries everything OnSessionEnd needs at fire time, captured
// when the session's ID was assigned by the SDK.
type endHookCtx struct {
	tenant     *Tenant
	sessionCtx context.Context
}

// serverEntry is the value stored in Handler.servers. createdAt records when
// getServer built the *mcp.Server, so reconciliation can apply a grace window
// before evicting a sessionless (still-initializing) server (Fix #9).
type serverEntry struct {
	createdAt time.Time
}

// tenantIdentifier returns a stable, log-safe identifier for the tenant
// derived from its credentials. SHA-256 of (service-token || "\x00" ||
// workspace), hex-encoded. Used for session-tenant binding to prevent
// stolen-session-ID reuse across tenants.
func tenantIdentifier(c tenant.Credentials) string {
	h := sha256.Sum256([]byte(c.ServiceToken + "\x00" + c.Workspace))
	return hex.EncodeToString(h[:])
}

// DeadOption is a future-extension hook for MarkSessionDead. None defined yet.
type DeadOption interface{ applyDead(*deadOptions) }

type deadOptions struct{}

// MarkSessionDead flags the current MCP session for closure after the
// in-flight tool call returns. The session is closed via *mcp.ServerSession.Close()
// before the SDK returns control to the HTTP handler. Subsequent requests
// bearing the same Mcp-Session-Id get HTTP 404; the client must reconnect.
//
// Use for sessions where downstream credentials have become invalid — for
// example, when a tool handler observes that the SuprSend API has rejected
// the tenant's credentials with HTTP 401. The internal/utils transport
// interceptor wires this to every HTTP 401 it observes from the management
// API.
//
// Safe to call from any context that derives from a request handled by this
// package. No-op if called outside such a context.
func MarkSessionDead(ctx context.Context, opts ...DeadOption) {
	if flag, ok := ctx.Value(deadSessionKey{}).(*atomic.Bool); ok {
		flag.Store(true)
	}
}

type deadSessionKey struct{}

// resolvedTenantKey carries the *Tenant from the auth middleware to getServer.
type resolvedTenantKey struct{}

// New returns a *Handler that serves MCP over streamable HTTP. Panics if
// opts.Resolver is nil.
//
// Capability profile applied to every per-tenant server: protocol-level
// logging is advertised explicitly, and the official SDK infers
// tools/list_changed from the AddTool calls the library makes. Resources and
// prompts capabilities are NOT advertised. Recovery is installed as outermost
// middleware by installSessionMiddleware (the SDK has no automatic recovery).
func New(opts Options) *Handler {
	if opts.Resolver == nil {
		panic("mcpserver.New: Resolver is required")
	}
	impl := opts.Implementation
	if impl == nil {
		impl = &mcp.Implementation{Name: "suprsend", Version: "dev"}
	}

	h := &Handler{
		opts:            opts,
		reconcileStop:   make(chan struct{}),
		reconcileDoneCh: make(chan struct{}),
	}

	getServer := func(r *http.Request) *mcp.Server {
		t, ok := r.Context().Value(resolvedTenantKey{}).(*Tenant)
		if !ok || t == nil {
			return nil
		}
		if h.closing.Load() {
			return nil
		}

		perCall := mergeServerOptions(opts.ServerOptions, defaultCapabilityProfile())
		tenantIdent := tenantIdentifier(t.Credentials)

		// Fix #11: carry the inbound initialize request's context VALUES
		// (resolved tenant, trace span, auth detail) into the session ctx,
		// but detach its cancellation so the session correctly outlives the
		// initialize request. context.WithoutCancel is Go 1.21+.
		sessionCtx := context.WithoutCancel(r.Context())
		safeCall("OnSessionStart", h.serverLogger(), func() {
			if h.opts.OnSessionStart != nil {
				sessionCtx = h.opts.OnSessionStart(sessionCtx, t)
			}
		})

		userGetSessionID := perCall.GetSessionID
		perCall.GetSessionID = func() string {
			var id string
			if userGetSessionID != nil {
				id = userGetSessionID()
			} else {
				id = randSessionID()
			}
			h.sessionTenants.Store(id, tenantIdent)
			h.sessionInfo.Store(id, &endHookCtx{tenant: t, sessionCtx: sessionCtx})
			return id
		}

		srv := mcp.NewServer(impl, perCall)
		h.servers.Store(srv, serverEntry{createdAt: time.Now()})

		// Fix #10: close the Shutdown TOCTOU window. The early h.closing check
		// above can pass and then Shutdown's server-walk can complete before we
		// Store, leaking this server (its session would never be closed). Re-
		// check after Store: if shutdown began, drop this server and close any
		// sessions that may already have attached, then refuse the session by
		// returning nil. Either Shutdown's Range sees the server (closes it) or
		// this re-check does; ss.Close is idempotent so both running is safe.
		if h.closing.Load() {
			h.servers.Delete(srv)
			for ss := range srv.Sessions() {
				_ = ss.Close()
			}
			return nil
		}

		installSessionMiddleware(srv, h, t, sessionCtx)
		for _, tool := range t.Tools {
			registerTool(srv, tool, h)
		}
		return srv
	}

	httpOpts := &mcp.StreamableHTTPOptions{}
	if opts.HTTPOptions != nil {
		*httpOpts = *opts.HTTPOptions
	} else {
		httpOpts.SessionTimeout = 30 * time.Minute
	}

	streamable := mcp.NewStreamableHTTPHandler(getServer, httpOpts)
	h.Handler = newAuthMiddleware(opts.Resolver, h, streamable)

	reconcileInterval := httpOpts.SessionTimeout / 2
	if reconcileInterval <= 0 {
		reconcileInterval = 15 * time.Minute
	}
	// Fix #9: grace window for sessionless server eviction == one reconcile
	// interval. A server stored by getServer but whose session id has not yet
	// been assigned by the SDK (GetSessionID fires after getServer returns)
	// must not be evicted by a reconcile tick landing in that gap.
	h.reconcileGrace = reconcileInterval

	go h.runReconciliation(reconcileInterval)

	return h
}

func (h *Handler) runReconciliation(interval time.Duration) {
	defer close(h.reconcileDoneCh)
	if interval <= 0 {
		interval = 15 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-h.reconcileStop:
			return
		case <-ticker.C:
			h.reconcileSessionTenants()
		}
	}
}

func (h *Handler) reconcileSessionTenants() {
	live := map[string]struct{}{}
	h.servers.Range(func(k, v any) bool {
		srv, ok := k.(*mcp.Server)
		if !ok || srv == nil {
			return true
		}
		anyLive := false
		for ss := range srv.Sessions() {
			if id := ss.ID(); id != "" {
				live[id] = struct{}{}
				anyLive = true
			}
		}
		if !anyLive {
			// Fix #9: don't evict a server that is younger than the grace
			// window — it may have been stored by getServer but not yet had
			// its session id assigned by the SDK (GetSessionID fires after
			// getServer returns). Evicting it here would leak the server
			// (never closed on Shutdown) and drop its hijack binding.
			entry, ok := v.(serverEntry)
			if ok && time.Since(entry.createdAt) <= h.reconcileGrace {
				return true
			}
			h.servers.Delete(k)
		}
		return true
	})
	h.sessionTenants.Range(func(k, _ any) bool {
		id, _ := k.(string)
		if _, ok := live[id]; !ok {
			h.sessionTenants.Delete(k)
		}
		return true
	})
	h.sessionInfo.Range(func(k, v any) bool {
		id, _ := k.(string)
		if _, ok := live[id]; ok {
			return true
		}
		h.sessionInfo.Delete(k)
		info, ok := v.(*endHookCtx)
		if !ok || info == nil || h.opts.OnSessionEnd == nil {
			return true
		}
		safeCall("OnSessionEnd", h.serverLogger(), func() {
			h.opts.OnSessionEnd(info.sessionCtx, info.tenant)
		})
		return true
	})
}

// safeCall invokes fn, recovering from panics so a misbehaving user hook
// cannot crash the request or reconciliation goroutine. Logs via logger if set.
func safeCall(hookName string, logger *slog.Logger, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			if logger != nil {
				logger.Error("mcpserver: panic in user hook",
					"hook", hookName,
					"panic", fmt.Sprint(r),
					"stack", string(debug.Stack()),
				)
			}
		}
	}()
	fn()
}

func (h *Handler) serverLogger() *slog.Logger {
	if h.opts.ServerOptions == nil {
		return nil
	}
	return h.opts.ServerOptions.Logger
}

func randSessionID() string {
	var b [16]byte
	if _, err := cryptorand.Read(b[:]); err != nil {
		panic(fmt.Sprintf("mcpserver: crypto/rand failure: %v", err))
	}
	return hex.EncodeToString(b[:])
}

// defaultCapabilityProfile returns the default *mcp.ServerOptions applied to
// every per-tenant server. Only logging capability is set here; tools.listChanged
// is inferred by the official SDK from the library's AddTool calls. This profile's
// Capabilities replaces base.Capabilities in mergeServerOptions.
func defaultCapabilityProfile() *mcp.ServerOptions {
	return &mcp.ServerOptions{
		Capabilities: &mcp.ServerCapabilities{
			Logging: &mcp.LoggingCapabilities{},
		},
	}
}

// mergeServerOptions — Profile wins for Capabilities; base wins for everything else.
func mergeServerOptions(base, profile *mcp.ServerOptions) *mcp.ServerOptions {
	out := &mcp.ServerOptions{}
	if base != nil {
		*out = *base
	}
	if profile != nil && profile.Capabilities != nil {
		out.Capabilities = profile.Capabilities
	}
	return out
}
