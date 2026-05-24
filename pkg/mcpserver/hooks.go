package mcpserver

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync/atomic"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/suprsend/cli/internal/mcpsdk/official"
	"github.com/suprsend/cli/pkg/mcpsdk"
	"github.com/suprsend/cli/pkg/tenant"
)

// installSessionMiddleware wires the per-session middleware chain on srv.
// One AddReceivingMiddleware call with multiple middlewares so ordering is
// deterministic and recovery is genuinely outermost (Bug-6 fix). The
// middleware chain becomes:
//
//	recovery(session(handler))
//
// The session middleware does:
//   - credential injection (tenant.WithCredentials on every method ctx)
//   - dead-session flag installation (atomic.Bool for MarkSessionDead)
//   - OnSessionStart context propagation (via mergeContextValues, Bug-8 fix)
//   - OnToolCall hook (if configured) — wraps tools/call only
//   - active-call tracking for Handler.Shutdown drain
//   - reactive close — after each tool call, if the dead flag is set, close
//     the *mcp.ServerSession.
//
// Session-tenant binding registration AND OnSessionStart invocation happen
// in getServer (Bugs 9 + 15), not here, because:
//   - the SDK assigns the session ID AFTER this function returns;
//   - sessionCtx must be in scope before GetSessionID fires so the endHookCtx
//     can be stored alongside the binding.
//
// OnSessionEnd is fired by the periodic reconciliation goroutine when it
// detects a session is no longer live (Bug-7 workaround — the SDK exposes
// no synchronous session-end hook).
func installSessionMiddleware(srv *mcp.Server, h *Handler, t *Tenant, sessionCtx context.Context) {
	deadFlag := new(atomic.Bool)

	// Single AddReceivingMiddleware call: first arg is OUTERMOST per the
	// SDK's documented ordering ("Middleware is applied from right to left,
	// so that the first one is executed first" — server.go:1346).
	// Verified semantics in shared.go addMiddleware (backward iteration of
	// the slice). Recovery wraps session middleware, which wraps the handler.
	srv.AddReceivingMiddleware(
		recoveryMiddleware(h.opts.ServerOptions),
		sessionMiddleware(h, t, sessionCtx, deadFlag),
	)
}

// sessionMiddleware returns the per-session middleware (credentials, dead-flag,
// hooks, reactive close, active-call tracking). Extracted so installSessionMiddleware
// can register it via a single AddReceivingMiddleware call alongside recovery.
func sessionMiddleware(h *Handler, t *Tenant, sessionCtx context.Context, deadFlag *atomic.Bool) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			// Merge session-scoped values into per-call ctx. Deadline +
			// cancellation stay with the per-call ctx; values fall back to
			// sessionCtx for keys not on the per-call ctx (real impl in
			// mergeContextValues, Bug-8 fix).
			ctx = mergeContextValues(ctx, sessionCtx)
			ctx = tenant.WithCredentials(ctx, t.Credentials)
			ctx = context.WithValue(ctx, deadSessionKey{}, deadFlag)

			if method != methodCallTool {
				return next(ctx, method, req)
			}

			// Tool call — track for graceful shutdown + observability hook.
			h.activeCalls.Add(1)
			defer h.activeCalls.Done()

			var after func(*mcpsdk.Result, error)
			if h.opts.OnToolCall != nil {
				toolName := toolNameFromRequest(req)
				safeCall("OnToolCall", h.serverLogger(), func() {
					ctx, after = h.opts.OnToolCall(ctx, toolName)
				})
			}

			result, err := next(ctx, method, req)

			if after != nil {
				safeCall("OnToolCall after-fn", h.serverLogger(), func() {
					after(resultToMcpsdk(result), err)
				})
			}

			// Reactive close — a downstream API call returned HTTP 401 and
			// utils.authExpiryTransport marked the session dead. Close the
			// underlying ServerSession so the SDK refuses further requests
			// on the same Mcp-Session-Id.
			if deadFlag.Load() {
				closeSessionFromRequest(req)
			}

			return result, err
		}
	}
}

// recoveryMiddleware catches panics from inner middleware / handlers and
// converts them to JSON-RPC errors so a single buggy handler does not crash
// the entire hosted process. Logs the panic + stack via the configured
// Logger if any.
func recoveryMiddleware(sopts *mcp.ServerOptions) mcp.Middleware {
	var logFn func(format string, args ...any)
	if sopts != nil && sopts.Logger != nil {
		logFn = func(format string, args ...any) { sopts.Logger.Error(fmt.Sprintf(format, args...)) }
	} else {
		logFn = func(format string, args ...any) {} // silent default; closed-source should set Logger
	}

	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (result mcp.Result, err error) {
			defer func() {
				if r := recover(); r != nil {
					logFn("panic in MCP handler method=%s panic=%v stack=%s", method, r, debug.Stack())
					// Override return values via named results.
					result = nil
					err = fmt.Errorf("internal server error: panic in handler %q", method)
				}
			}()
			return next(ctx, method, req)
		}
	}
}

// registerTool registers a single mcpsdk.Tool on srv via the official
// adapter. Extracted into hooks.go because it'll later grow to install
// per-tool tracing or per-tool middleware as observability needs grow.
func registerTool(srv *mcp.Server, t *mcpsdk.Tool, h *Handler) {
	official.Register(srv, t)
}

// methodCallTool is the JSON-RPC method name for tool invocation. Mirrors
// the SDK's internal constant (mcp/protocol.go).
const methodCallTool = "tools/call"

// mergeContextValues returns a context whose Deadline/Done/Err come from
// child (per-call cancellation governs) but whose Value lookups fall back
// to parent (the session-scoped ctx from OnSessionStart) for any key not
// present on child.
//
// This is what makes OnSessionStart actually useful: a closed-source resolver
// can stash a span context, request ID, tenant logger, etc. on the returned
// session ctx, and every per-call handler sees those values via standard
// context.Value lookups.
func mergeContextValues(child, parent context.Context) context.Context {
	if parent == nil {
		return child
	}
	return &mergedContext{Context: child, parent: parent}
}

type mergedContext struct {
	context.Context        // child — provides Deadline, Done, Err
	parent          context.Context
}

// Value falls back to parent when child has no value for the key. This lets
// session-scoped values (set by OnSessionStart on the parent) be readable
// from per-call ctx.Value calls.
func (m *mergedContext) Value(key any) any {
	if v := m.Context.Value(key); v != nil {
		return v
	}
	return m.parent.Value(key)
}

// toolNameFromRequest extracts the public tool name from a CallToolRequest.
// req is concretely *mcp.CallToolRequest = ServerRequest[*CallToolParamsRaw];
// the tool name is at req.Params.Name (mcp/protocol.go:58).
func toolNameFromRequest(req mcp.Request) string {
	if r, ok := req.(*mcp.CallToolRequest); ok && r.Params != nil {
		return r.Params.Name
	}
	return ""
}

// resultToMcpsdk converts the SDK's mcp.Result (concretely *CallToolResult
// for tools/call) back into our *mcpsdk.Result so the OnToolCall after-fn
// gets a runtime-agnostic value. Mirrors internal/mcpsdk/official.toCallToolResult.
func resultToMcpsdk(r mcp.Result) *mcpsdk.Result {
	ctr, ok := r.(*mcp.CallToolResult)
	if !ok || ctr == nil {
		return nil
	}
	out := &mcpsdk.Result{IsError: ctr.IsError, Structured: ctr.StructuredContent}
	for _, c := range ctr.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			out.Text = tc.Text
			break
		}
	}
	return out
}

// closeSessionFromRequest looks up the ServerSession associated with req
// and calls Close on it. The CallToolRequest's Session field exposes the
// *ServerSession (mcp/shared.go ServerRequest.Session).
func closeSessionFromRequest(req mcp.Request) {
	if r, ok := req.(*mcp.CallToolRequest); ok && r.Session != nil {
		_ = r.Session.Close() // idempotent + concurrency-safe per SDK
	}
}
