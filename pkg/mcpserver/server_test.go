package mcpserver_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/suprsend/cli/pkg/mcpsdk"
	"github.com/suprsend/cli/pkg/mcpserver"
	"github.com/suprsend/cli/pkg/tenant"
)

func emptyObjectSchema() *jsonschema.Schema {
	return &jsonschema.Schema{Type: "object"}
}

func tenantNamedTool(name string) *mcpsdk.Tool {
	return &mcpsdk.Tool{
		Name:        name,
		Description: "Test tool exclusive to one tenant.",
		InputSchema: emptyObjectSchema(),
		Handler: func(_ context.Context, _ mcpsdk.Args) (mcpsdk.Result, error) {
			return mcpsdk.Result{Text: "ok:" + name}, nil
		},
	}
}

func TestPerTenantToolListings(t *testing.T) {
	fake := mcpserver.NewFakeResolver()
	fake.Add("token-a", &mcpserver.Tenant{
		Credentials: tenant.Credentials{ServiceToken: "token-a", Workspace: "ws-a"},
		Tools:       []*mcpsdk.Tool{tenantNamedTool("alpha_tool")},
	})
	fake.Add("token-b", &mcpserver.Tenant{
		Credentials: tenant.Credentials{ServiceToken: "token-b", Workspace: "ws-b"},
		Tools:       []*mcpsdk.Tool{tenantNamedTool("beta_tool")},
	})

	handler := mcpserver.New(mcpserver.Options{
		Resolver:       fake,
		Implementation: &mcp.Implementation{Name: "test-host", Version: "0.0.1"},
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	ctx := context.Background()

	tools := listToolsForToken(t, ctx, srv.URL, "token-a")
	if got := toolNames(tools); !contains(got, "alpha_tool") || contains(got, "beta_tool") {
		t.Errorf("tenant a sees %v, want alpha_tool only", got)
	}

	tools = listToolsForToken(t, ctx, srv.URL, "token-b")
	if got := toolNames(tools); !contains(got, "beta_tool") || contains(got, "alpha_tool") {
		t.Errorf("tenant b sees %v, want beta_tool only", got)
	}
}

func TestUnknownTokenFails(t *testing.T) {
	fake := mcpserver.NewFakeResolver()
	handler := mcpserver.New(mcpserver.Options{Resolver: fake})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	ctx := context.Background()
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.1"}, nil)
	tr := &mcp.StreamableClientTransport{
		Endpoint:   srv.URL,
		HTTPClient: bearerClient("unknown-token"),
	}
	if _, err := client.Connect(ctx, tr, nil); err == nil {
		t.Fatal("expected connect failure for unknown token")
	}
}

func listToolsForToken(t *testing.T, ctx context.Context, endpoint, token string) []*mcp.Tool {
	t.Helper()
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.1"}, nil)
	tr := &mcp.StreamableClientTransport{
		Endpoint:   endpoint,
		HTTPClient: bearerClient(token),
	}
	session, err := client.Connect(ctx, tr, nil)
	if err != nil {
		t.Fatalf("connect %q: %v", token, err)
	}
	defer session.Close()

	res, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools %q: %v", token, err)
	}
	return res.Tools
}

func toolNames(tools []*mcp.Tool) []string {
	out := make([]string, len(tools))
	for i, t := range tools {
		out[i] = t.Name
	}
	return out
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

type bearerTransport struct {
	token string
	base  http.RoundTripper
}

func (b *bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+b.token)
	return b.base.RoundTrip(req)
}

func bearerClient(token string) *http.Client {
	return &http.Client{Transport: &bearerTransport{token: token, base: http.DefaultTransport}}
}

func TestUnauthorizedReturnsHTTP401(t *testing.T) {
	fake := mcpserver.NewFakeResolver() // no tenants added
	handler := mcpserver.New(mcpserver.Options{Resolver: fake})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.URL, http.NoBody)
	req.Header.Set("Authorization", "Bearer wrong-token")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
	if got := resp.Header.Get("WWW-Authenticate"); got == "" {
		t.Error("WWW-Authenticate header missing")
	}
}

func TestForbiddenReturnsHTTP403(t *testing.T) {
	resolver := &alwaysForbiddenResolver{}
	handler := mcpserver.New(mcpserver.Options{Resolver: resolver})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.URL, http.NoBody)
	req.Header.Set("Authorization", "Bearer anything")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want 403", resp.StatusCode)
	}
}

type alwaysForbiddenResolver struct{}

func (alwaysForbiddenResolver) Resolve(_ context.Context, _ *http.Request) (*mcpserver.Tenant, error) {
	return nil, fmt.Errorf("policy says no: %w", mcpserver.ErrForbidden)
}

func TestReactiveSessionCloseOn401InHandler(t *testing.T) {
	// Tool handler that immediately marks the session dead, simulating a
	// downstream API returning 401 mid-call.
	deadTool := &mcpsdk.Tool{
		Name:        "fail_with_auth",
		Description: "Simulates a downstream 401.",
		InputSchema: emptyObjectSchema(),
		Handler: func(ctx context.Context, _ mcpsdk.Args) (mcpsdk.Result, error) {
			mcpserver.MarkSessionDead(ctx)
			return mcpsdk.Result{Text: "auth expired", IsError: true}, nil
		},
	}
	fake := mcpserver.NewFakeResolver()
	fake.Add("tok", &mcpserver.Tenant{
		Credentials: tenant.Credentials{ServiceToken: "tok", Workspace: "ws"},
		Tools:       []*mcpsdk.Tool{deadTool},
	})
	handler := mcpserver.New(mcpserver.Options{Resolver: fake})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	ctx := context.Background()
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0.0.1"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: srv.URL, HTTPClient: bearerClient("tok")}, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	// First call — marks session dead. Result includes IsError=true.
	if _, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "fail_with_auth"}); err != nil {
		t.Fatalf("first call: %v", err)
	}

	// Second call on the SAME session — must fail because the session was
	// closed by the reactive-close mechanism.
	if _, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "fail_with_auth"}); err == nil {
		t.Error("second call on dead session unexpectedly succeeded")
	}
}

func TestGracefulShutdownDrainsInFlight(t *testing.T) {
	slowTool := &mcpsdk.Tool{
		Name:        "slow",
		Description: "Sleeps to simulate in-flight work.",
		InputSchema: emptyObjectSchema(),
		Handler: func(ctx context.Context, _ mcpsdk.Args) (mcpsdk.Result, error) {
			select {
			case <-time.After(200 * time.Millisecond):
				return mcpsdk.Result{Text: "done"}, nil
			case <-ctx.Done():
				return mcpsdk.Result{Text: "cancelled", IsError: true}, ctx.Err()
			}
		},
	}
	fake := mcpserver.NewFakeResolver()
	fake.Add("tok", &mcpserver.Tenant{
		Credentials: tenant.Credentials{ServiceToken: "tok", Workspace: "ws"},
		Tools:       []*mcpsdk.Tool{slowTool},
	})
	handler := mcpserver.New(mcpserver.Options{Resolver: fake})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	ctx := context.Background()
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0.0.1"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: srv.URL, HTTPClient: bearerClient("tok")}, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer session.Close()

	// Fire a slow call, then immediately initiate shutdown.
	callDone := make(chan error, 1)
	go func() {
		_, e := session.CallTool(ctx, &mcp.CallToolParams{Name: "slow"})
		callDone <- e
	}()

	time.Sleep(20 * time.Millisecond) // let the call start
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := handler.Shutdown(shutdownCtx); err != nil {
		t.Errorf("Shutdown: %v", err)
	}

	select {
	case err := <-callDone:
		if err != nil {
			t.Errorf("in-flight call did not complete cleanly: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("in-flight call did not return after Shutdown")
	}
}

func TestSessionHijackRejected(t *testing.T) {
	// Two tenants with valid tokens. Establish a session as tenant A, then
	// reuse the same Mcp-Session-Id with tenant B's token — expect HTTP 403.
	fake := mcpserver.NewFakeResolver()
	fake.Add("token-a", &mcpserver.Tenant{
		Credentials: tenant.Credentials{ServiceToken: "token-a", Workspace: "ws-a"},
		Tools:       []*mcpsdk.Tool{tenantNamedTool("alpha_tool")},
	})
	fake.Add("token-b", &mcpserver.Tenant{
		Credentials: tenant.Credentials{ServiceToken: "token-b", Workspace: "ws-b"},
		Tools:       []*mcpsdk.Tool{tenantNamedTool("beta_tool")},
	})
	handler := mcpserver.New(mcpserver.Options{Resolver: fake})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	// Establish a session as tenant A; capture the Mcp-Session-Id from the response.
	initReq, _ := http.NewRequest(http.MethodPost, srv.URL, strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"0"}}}`))
	initReq.Header.Set("Authorization", "Bearer token-a")
	initReq.Header.Set("Content-Type", "application/json")
	initReq.Header.Set("Accept", "application/json, text/event-stream")
	resp, err := http.DefaultClient.Do(initReq)
	if err != nil {
		t.Fatalf("initialize as A: %v", err)
	}
	// Drain the SSE response body so the connection can be cleanly closed.
	// Without this, the test occasionally leaks a goroutine that races
	// against test teardown — bare Close() on an unread event-stream body
	// leaves the underlying connection in an indeterminate state.
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	sessionID := resp.Header.Get("Mcp-Session-Id")
	if sessionID == "" {
		t.Skip("server did not assign Mcp-Session-Id; check stateful mode")
	}

	// Reuse the session ID with tenant B's bearer token — must 403.
	hijackReq, _ := http.NewRequest(http.MethodPost, srv.URL, strings.NewReader(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`))
	hijackReq.Header.Set("Authorization", "Bearer token-b")
	hijackReq.Header.Set("Mcp-Session-Id", sessionID)
	hijackReq.Header.Set("Content-Type", "application/json")
	hijackReq.Header.Set("Accept", "application/json, text/event-stream")
	hijack, err := http.DefaultClient.Do(hijackReq)
	if err != nil {
		t.Fatalf("hijack request: %v", err)
	}
	defer hijack.Body.Close()
	if hijack.StatusCode != http.StatusForbidden {
		t.Errorf("session-hijack status = %d, want 403", hijack.StatusCode)
	}
}

func TestPanicInHandlerReturnsErrorNotCrash(t *testing.T) {
	panicTool := &mcpsdk.Tool{
		Name:        "panic_tool",
		Description: "Deliberately panics — verifies recovery middleware.",
		InputSchema: emptyObjectSchema(),
		Handler: func(_ context.Context, _ mcpsdk.Args) (mcpsdk.Result, error) {
			panic("intentional test panic")
		},
	}
	fake := mcpserver.NewFakeResolver()
	fake.Add("tok", &mcpserver.Tenant{
		Credentials: tenant.Credentials{ServiceToken: "tok", Workspace: "ws"},
		Tools:       []*mcpsdk.Tool{panicTool},
	})
	handler := mcpserver.New(mcpserver.Options{Resolver: fake})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	ctx := context.Background()
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0.0.1"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: srv.URL, HTTPClient: bearerClient("tok")}, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer session.Close()

	// Call the panicking tool — recovery middleware converts the panic to a
	// returned error so the SDK surfaces it as a JSON-RPC error.
	if _, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "panic_tool"}); err == nil {
		t.Error("expected error from panicking tool, got nil")
	}

	// Second call on the SAME session — the point of this assertion is that
	// the server process is still alive after the first panic (no crash).
	// We expect another error (the tool panics again) — but receiving ANY
	// response, error or not, proves the goroutine + server survived.
	if _, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "panic_tool"}); err == nil {
		t.Error("second panicking call returned nil error; recovery middleware should report an error")
	}
}

func TestObservabilityHooksFire(t *testing.T) {
	var (
		startCount atomic.Int32
		endCount   atomic.Int32
		callCount  atomic.Int32
	)
	echoTool := &mcpsdk.Tool{
		Name:        "echo",
		Description: "Returns ok.",
		InputSchema: emptyObjectSchema(),
		Handler: func(_ context.Context, _ mcpsdk.Args) (mcpsdk.Result, error) {
			return mcpsdk.Result{Text: "ok"}, nil
		},
	}
	fake := mcpserver.NewFakeResolver()
	fake.Add("tok", &mcpserver.Tenant{
		Credentials: tenant.Credentials{ServiceToken: "tok", Workspace: "ws"},
		Tools:       []*mcpsdk.Tool{echoTool},
	})
	// OnSessionEnd fires from the background reconciliation goroutine, which
	// runs at SessionTimeout/2 intervals. For this test we want it to fire
	// within a few hundred ms of session.Close(), so pass a very small
	// SessionTimeout — reconciliation will run ~every 50 ms.
	handler := mcpserver.New(mcpserver.Options{
		Resolver: fake,
		HTTPOptions: &mcp.StreamableHTTPOptions{
			SessionTimeout: 100 * time.Millisecond, // reconciliation interval = 50 ms
		},
		OnSessionStart: func(ctx context.Context, _ *mcpserver.Tenant) context.Context {
			startCount.Add(1)
			return ctx
		},
		OnSessionEnd: func(_ context.Context, _ *mcpserver.Tenant) {
			endCount.Add(1)
		},
		OnToolCall: func(ctx context.Context, _ string) (context.Context, func(*mcpsdk.Result, error)) {
			return ctx, func(_ *mcpsdk.Result, _ error) { callCount.Add(1) }
		},
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	ctx := context.Background()
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0.0.1"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: srv.URL, HTTPClient: bearerClient("tok")}, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if _, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "echo"}); err != nil {
		t.Fatalf("call: %v", err)
	}
	if err := session.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	// Wait for SessionTimeout + at least one reconcile tick (50 ms each, 100 ms
	// SessionTimeout → SDK reaps the session, then reconciliation evicts our
	// sessionInfo + fires OnSessionEnd). 300 ms is comfortable headroom.
	time.Sleep(300 * time.Millisecond)

	if startCount.Load() != 1 {
		t.Errorf("OnSessionStart fired %d times, want 1", startCount.Load())
	}
	if endCount.Load() != 1 {
		t.Errorf("OnSessionEnd fired %d times, want 1", endCount.Load())
	}
	if callCount.Load() != 1 {
		t.Errorf("OnToolCall after-fn fired %d times, want 1", callCount.Load())
	}
}

func TestBuildTenantTools_StaticOnly(t *testing.T) {
	ctx := tenant.WithCredentials(context.Background(), tenant.Credentials{
		ServiceToken:      "tok",
		Workspace:         "staging",
		WorkflowsSelector: "none",
		EventsSelector:    "none",
	})
	tools, err := mcpserver.BuildTenantTools(ctx)
	if err != nil {
		t.Fatalf("BuildTenantTools: %v", err)
	}
	if len(tools) == 0 {
		t.Fatal("expected at least the static tool set, got 0")
	}
	// Sanity: a known static tool name appears.
	if !containsToolNamed(tools, "get_suprsend_tenant") {
		t.Errorf("static tool get_suprsend_tenant missing from %v", toolNamesMcpsdk(tools))
	}
}

func TestBuildTenantTools_MissingCredentials(t *testing.T) {
	if _, err := mcpserver.BuildTenantTools(context.Background()); err == nil {
		t.Fatal("expected error when ctx has no tenant credentials")
	}
}

func containsToolNamed(tools []*mcpsdk.Tool, name string) bool {
	for _, t := range tools {
		if t.Name == name {
			return true
		}
	}
	return false
}

func toolNamesMcpsdk(tools []*mcpsdk.Tool) []string {
	out := make([]string, len(tools))
	for i, t := range tools {
		out[i] = t.Name
	}
	return out
}
