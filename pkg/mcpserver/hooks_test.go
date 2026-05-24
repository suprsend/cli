package mcpserver

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestRecoveryMiddleware_CatchesPanic — wrap a panicking next, assert the
// returned err is non-nil, no panic propagates.
func TestRecoveryMiddleware_CatchesPanic(t *testing.T) {
	mw := recoveryMiddleware(nil) // nil ServerOptions → silent logger
	next := mcp.MethodHandler(func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
		panic("boom")
	})

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("recoveryMiddleware let a panic escape: %v", r)
		}
	}()

	result, err := mw(next)(context.Background(), "tools/call", nil)
	if result != nil {
		t.Errorf("expected nil result after panic, got %v", result)
	}
	if err == nil {
		t.Fatal("expected non-nil error after panic, got nil")
	}
	if !strings.Contains(err.Error(), "internal server error") {
		t.Errorf("expected error to mention 'internal server error', got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "tools/call") {
		t.Errorf("expected error to mention method name 'tools/call', got %q", err.Error())
	}
}

// TestRecoveryMiddleware_NoPanic_PassesThrough — wrap a normal next, assert
// result/err pass through unchanged.
func TestRecoveryMiddleware_NoPanic_PassesThrough(t *testing.T) {
	mw := recoveryMiddleware(nil)
	wantResult := &mcp.CallToolResult{IsError: false}
	wantErr := errors.New("downstream failure")
	next := mcp.MethodHandler(func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
		return wantResult, wantErr
	})

	gotResult, gotErr := mw(next)(context.Background(), "tools/call", nil)
	if gotResult != wantResult {
		t.Errorf("result mutated by recovery middleware: want %p got %p", wantResult, gotResult)
	}
	if gotErr != wantErr {
		t.Errorf("err mutated by recovery middleware: want %v got %v", wantErr, gotErr)
	}
}

type ctxKey string

// TestMergeContextValues_ChildOverridesParent — child has key K = v1, parent
// has K = v2, lookup returns v1 (child wins).
func TestMergeContextValues_ChildOverridesParent(t *testing.T) {
	k := ctxKey("k")
	parent := context.WithValue(context.Background(), k, "parent-value")
	child := context.WithValue(context.Background(), k, "child-value")

	merged := mergeContextValues(child, parent)
	if got := merged.Value(k); got != "child-value" {
		t.Errorf("child should win: want %q got %v", "child-value", got)
	}
}

// TestMergeContextValues_FallsBackToParent — child has no key K; parent has
// K = v; lookup returns v.
func TestMergeContextValues_FallsBackToParent(t *testing.T) {
	k := ctxKey("only-on-parent")
	parent := context.WithValue(context.Background(), k, "parent-value")
	child := context.Background() // no value for k

	merged := mergeContextValues(child, parent)
	if got := merged.Value(k); got != "parent-value" {
		t.Errorf("expected fallback to parent: want %q got %v", "parent-value", got)
	}
}

// TestMergeContextValues_NilParent — nil parent returns child unchanged.
func TestMergeContextValues_NilParent(t *testing.T) {
	child := context.Background()
	merged := mergeContextValues(child, nil)
	if merged != child {
		t.Errorf("nil parent should return child unchanged: want %p got %p", child, merged)
	}
}

// TestMergeContextValues_CancelFromChild — child cancelled, merged.Done() is
// closed; parent's cancellation does not affect merged.
func TestMergeContextValues_CancelFromChild(t *testing.T) {
	parent := context.Background() // never cancelled
	child, cancel := context.WithCancel(context.Background())
	merged := mergeContextValues(child, parent)

	select {
	case <-merged.Done():
		t.Fatal("merged was cancelled before child cancel called")
	default:
	}

	cancel()

	select {
	case <-merged.Done():
		// expected
	case <-time.After(time.Second):
		t.Fatal("merged.Done() never closed after child cancellation")
	}

	if err := merged.Err(); err == nil {
		t.Errorf("merged.Err() should be non-nil after child cancellation, got nil")
	}
}

// TestToolNameFromRequest_WithCallToolRequest — *CallToolRequest with Params
// set yields the tool name.
func TestToolNameFromRequest_WithCallToolRequest(t *testing.T) {
	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: "echo"}}
	if got := toolNameFromRequest(req); got != "echo" {
		t.Errorf("toolNameFromRequest: want %q got %q", "echo", got)
	}
}

// TestToolNameFromRequest_NilParams — *CallToolRequest with nil Params yields "".
func TestToolNameFromRequest_NilParams(t *testing.T) {
	req := &mcp.CallToolRequest{Params: nil}
	if got := toolNameFromRequest(req); got != "" {
		t.Errorf("toolNameFromRequest with nil Params: want %q got %q", "", got)
	}
}

// TestToolNameFromRequest_NilRequest — nil request yields "".
func TestToolNameFromRequest_NilRequest(t *testing.T) {
	if got := toolNameFromRequest(nil); got != "" {
		t.Errorf("toolNameFromRequest(nil): want %q got %q", "", got)
	}
}

// TestResultToMcpsdk_TextContent — CallToolResult with TextContent maps to
// mcpsdk.Result.Text.
func TestResultToMcpsdk_TextContent(t *testing.T) {
	ctr := &mcp.CallToolResult{
		Content:           []mcp.Content{&mcp.TextContent{Text: "hello"}},
		IsError:           true,
		StructuredContent: map[string]any{"k": "v"},
	}
	out := resultToMcpsdk(ctr)
	if out == nil {
		t.Fatal("resultToMcpsdk returned nil")
	}
	if out.Text != "hello" {
		t.Errorf("Text: want %q got %q", "hello", out.Text)
	}
	if !out.IsError {
		t.Errorf("IsError should be true")
	}
	if m, ok := out.Structured.(map[string]any); !ok || m["k"] != "v" {
		t.Errorf("Structured not propagated: got %v", out.Structured)
	}
}

// TestResultToMcpsdk_NilResult — non-CallToolResult input returns nil.
func TestResultToMcpsdk_NilResult(t *testing.T) {
	if got := resultToMcpsdk(nil); got != nil {
		t.Errorf("resultToMcpsdk(nil): want nil got %v", got)
	}
}
