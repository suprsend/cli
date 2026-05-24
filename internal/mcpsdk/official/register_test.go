package official_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/suprsend/cli/internal/mcpsdk/official"
	"github.com/suprsend/cli/pkg/mcpsdk"
)

func TestRegister_RoundTrip(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "0.0.1"}, nil)

	tool := &mcpsdk.Tool{
		Name:        "echo",
		Description: "Echo back the input string.",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"text": {Type: "string"},
			},
		},
		Meta: map[string]any{
			"ui": map[string]any{"resourceUri": "ui://echo"},
		},
		Annotations: mcpsdk.Annotations{ReadOnlyHint: true},
		Handler: func(_ context.Context, args mcpsdk.Args) (mcpsdk.Result, error) {
			text, err := args.RequireString("text")
			if err != nil {
				return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
			}
			return mcpsdk.Result{Text: strings.ToUpper(text)}, nil
		},
	}
	official.Register(server, tool)

	clientT, serverT := mcp.NewInMemoryTransports()
	go func() { _, _ = server.Connect(ctx, serverT, nil) }()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.1"}, nil)
	session, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer session.Close()

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	if len(tools.Tools) != 1 || tools.Tools[0].Name != "echo" {
		t.Fatalf("ListTools: %+v", tools.Tools)
	}
	if tools.Tools[0].Meta["ui"] == nil {
		t.Errorf("_meta.ui not present on listed tool")
	}
	if tools.Tools[0].Annotations == nil || !tools.Tools[0].Annotations.ReadOnlyHint {
		t.Errorf("ReadOnlyHint not set: %+v", tools.Tools[0].Annotations)
	}

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "echo",
		Arguments: map[string]any{"text": "hello"},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if len(res.Content) == 0 {
		t.Fatal("empty content")
	}
	tc, ok := res.Content[0].(*mcp.TextContent)
	if !ok || tc.Text != "HELLO" {
		t.Errorf("unexpected content: %+v", res.Content[0])
	}
}
