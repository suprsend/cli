package mcpserver

import (
	"context"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Shutdown stops accepting new sessions, waits for in-flight tool calls to
// finish (or for ctx to expire), then closes existing MCP sessions cleanly.
// Safe to call multiple times; subsequent calls return nil immediately.
//
// The caller is expected to invoke this BEFORE closing the HTTP listener:
//
//	signal := make(chan os.Signal, 1)
//	signal.Notify(signal, syscall.SIGTERM)
//	<-signal
//	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//	if err := mcpHandler.Shutdown(shutdownCtx); err != nil {
//	    log.Printf("MCP shutdown: %v", err)
//	}
//	httpServer.Shutdown(shutdownCtx)
func (h *Handler) Shutdown(ctx context.Context) error {
	var shutdownErr error
	h.shutdownOnce.Do(func() {
		// 1. Stop accepting new sessions. The auth middleware checks this
		//    flag and returns 503 for fresh requests; getServer also checks
		//    and returns nil (→ 400 from the SDK) for any race.
		h.closing.Store(true)

		// 2. Stop the reconciliation goroutine and wait for it to exit so
		//    we don't race against it during the session-walk below.
		close(h.reconcileStop)
		<-h.reconcileDoneCh

		// 3. Wait for in-flight tool calls to finish.
		done := make(chan struct{})
		go func() { h.activeCalls.Wait(); close(done) }()
		select {
		case <-done:
		case <-ctx.Done():
			shutdownErr = errors.Join(errors.New("mcpserver: shutdown ctx expired with in-flight tool calls"), ctx.Err())
			// Fall through to step 4 anyway — close what we can.
		}

		// 4. Close existing sessions cleanly across all per-tenant servers.
		//    h.servers holds every *mcp.Server returned by getServer; each
		//    server's Sessions() iterator yields its live sessions.
		//    ServerSession.Close is idempotent + concurrency-safe (verified
		//    server.go:1508).
		h.servers.Range(func(key, _ any) bool {
			srv, ok := key.(*mcp.Server)
			if !ok || srv == nil {
				return true
			}
			for ss := range srv.Sessions() {
				if err := ss.Close(); err != nil && shutdownErr == nil {
					shutdownErr = err
				}
			}
			h.servers.Delete(key)
			return true
		})
	})
	return shutdownErr
}
