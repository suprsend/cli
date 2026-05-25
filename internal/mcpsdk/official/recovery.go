package official

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RecoveryMiddleware catches panics from inner middleware / handlers and
// converts them to JSON-RPC errors so a single buggy handler does not crash
// the entire server process. Logs the panic + stack via the supplied logger.
//
// This mirrors pkg/mcpserver's recoveryMiddleware but is exported here so the
// single-tenant CLI server (internal/commands/startMcpServer.go) can install
// recovery on its own *mcp.Server without importing the heavier pkg/mcpserver.
//
// Register it OUTERMOST via mcpServer.AddReceivingMiddleware so it wraps every
// receiving method handler. The middleware uses NAMED returns so the deferred
// recover() can override the result/error after a panic.
func RecoveryMiddleware(logger *slog.Logger) mcp.Middleware {
	logFn := func(format string, args ...any) {}
	if logger != nil {
		logFn = func(format string, args ...any) { logger.Error(fmt.Sprintf(format, args...)) }
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
