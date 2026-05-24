// Package legacy adapts mcpsdk.Tool to the existing mark3labs/mcp-go runtime.
// Used by the CLI's stdio transport during the migration. Removed in phase 6
// once the CLI flips to the official adapter.
package legacy

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/suprsend/cli/pkg/mcpsdk"
)

// Register registers a single mcpsdk.Tool against a mark3labs server.
func Register(s *server.MCPServer, t *mcpsdk.Tool) {
	schemaBytes, err := json.Marshal(t.InputSchema)
	if err != nil {
		// Schema marshal failure is a programming error in the tool definition,
		// not a runtime condition the caller can recover from.
		panic("mcpsdk/legacy: marshal input schema for " + t.Name + ": " + err.Error())
	}
	mcpTool := mcp.NewToolWithRawSchema(t.Name, t.Description, schemaBytes)
	if t.Meta != nil {
		mcpTool.Meta = &mcp.Meta{AdditionalFields: t.Meta}
	}
	mcpTool.Annotations = convertAnnotations(t.Annotations)

	s.AddTool(mcpTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw, _ := json.Marshal(req.GetArguments())
		args, err := mcpsdk.NewArgs(raw)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		res, err := t.Handler(ctx, args)
		if err != nil {
			return nil, err
		}
		if res.IsError {
			return mcp.NewToolResultError(res.Text), nil
		}
		if res.Structured != nil {
			payload, mErr := json.Marshal(res.Structured)
			if mErr != nil {
				return mcp.NewToolResultError(mErr.Error()), nil
			}
			return mcp.NewToolResultStructured(res.Structured, string(payload)), nil
		}
		return mcp.NewToolResultText(res.Text), nil
	})
}

func convertAnnotations(a mcpsdk.Annotations) mcp.ToolAnnotation {
	return mcp.ToolAnnotation{
		Title:           a.Title,
		ReadOnlyHint:    boolPtr(a.ReadOnlyHint),
		IdempotentHint:  boolPtr(a.IdempotentHint),
		DestructiveHint: boolPtr(a.DestructiveHint),
		OpenWorldHint:   boolPtr(a.OpenWorldHint),
	}
}

func boolPtr(b bool) *bool { return &b }
