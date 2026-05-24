// Package official adapts mcpsdk.Tool to the modelcontextprotocol/go-sdk
// runtime. Call Register once per tool against a constructed *mcp.Server.
package official

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/suprsend/cli/pkg/mcpsdk"
)

// Register registers a single mcpsdk.Tool against the given official-SDK
// server. Handler conversion translates Args and Result back and forth.
func Register(server *mcp.Server, t *mcpsdk.Tool) {
	server.AddTool(&mcp.Tool{
		Meta:        mcp.Meta(t.Meta),
		Name:        t.Name,
		Description: t.Description,
		InputSchema: t.InputSchema,
		Annotations: convertAnnotations(t.Annotations),
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := mcpsdk.NewArgs(req.Params.Arguments)
		if err != nil {
			return errorResult(err.Error()), nil
		}
		res, err := t.Handler(ctx, args)
		if err != nil {
			return nil, err
		}
		return toCallToolResult(res), nil
	})
}

func convertAnnotations(a mcpsdk.Annotations) *mcp.ToolAnnotations {
	if (a == mcpsdk.Annotations{}) {
		return nil
	}
	// Verified mcp/protocol.go:1357-1381:
	//   DestructiveHint *bool, OpenWorldHint *bool  (pointer — nil = unset)
	//   ReadOnlyHint    bool,  IdempotentHint bool  (value)
	// boolPtr wraps the bool so we emit either &true or &false (never nil)
	// once a tool author has set the field on mcpsdk.Annotations.
	return &mcp.ToolAnnotations{
		Title:           a.Title,
		ReadOnlyHint:    a.ReadOnlyHint,
		IdempotentHint:  a.IdempotentHint,
		DestructiveHint: boolPtr(a.DestructiveHint),
		OpenWorldHint:   boolPtr(a.OpenWorldHint),
	}
}

func boolPtr(b bool) *bool { return &b }

func toCallToolResult(r mcpsdk.Result) *mcp.CallToolResult {
	out := &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: r.Text}},
		IsError: r.IsError,
	}
	if r.Structured != nil {
		out.StructuredContent = r.Structured
	}
	return out
}

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
		IsError: true,
	}
}
