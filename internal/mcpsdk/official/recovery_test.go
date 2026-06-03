package official

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestRecoveryMiddleware_CatchesPanic — wrap a panicking next, assert the
// returned err is non-nil and no panic escapes (the CLI stdio process survives).
func TestRecoveryMiddleware_CatchesPanic(t *testing.T) {
	mw := RecoveryMiddleware(nil) // nil logger → silent
	next := mcp.MethodHandler(func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
		panic("boom")
	})

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("RecoveryMiddleware let a panic escape: %v", r)
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
	mw := RecoveryMiddleware(nil)
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
