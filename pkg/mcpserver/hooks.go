package mcpserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/suprsend/cli/pkg/mcpsdk"
)

// installSessionMiddleware — STUB. Real implementation lands in Task 3.1c.
func installSessionMiddleware(srv *mcp.Server, h *Handler, t *Tenant, sessionCtx context.Context) {
	panic("mcpserver: installSessionMiddleware not implemented yet (Task 3.1c)")
}

// registerTool — STUB. Real implementation lands in Task 3.1c.
func registerTool(srv *mcp.Server, t *mcpsdk.Tool, h *Handler) {
	panic("mcpserver: registerTool not implemented yet (Task 3.1c)")
}
