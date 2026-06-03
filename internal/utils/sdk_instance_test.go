package utils

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/suprsend/cli/mgmnt"
	"github.com/suprsend/cli/pkg/tenant"
)

// roundTripperFunc lets a test return a canned response from an http.RoundTripper.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// resetSDKGlobals clears the package-level singletons that tests mutate so
// each test starts from a clean slate. Call via t.Cleanup.
func resetSDKGlobals(t *testing.T) {
	t.Helper()
	prevSDK := SDKInstance
	prevMark := MarkSessionDead
	mgmntCacheMu.Lock()
	prevCache := mgmntCache
	mgmntCache = map[mgmntCacheKey]*mgmnt.SS_MgmntClient{}
	mgmntCacheMu.Unlock()
	t.Cleanup(func() {
		mgmntCacheMu.Lock()
		mgmntCache = prevCache
		mgmntCacheMu.Unlock()
		SDKInstance = prevSDK
		MarkSessionDead = prevMark
	})
}

func TestMgmntClientFor_NoCredentials_ReturnsSingleton(t *testing.T) {
	resetSDKGlobals(t)
	sentinel := mgmnt.NewClientWithUrls("tok", "https://hub.example.com", "https://mgmnt.example.com/", false)
	SDKInstance = sentinel

	got := MgmntClientFor(context.Background())
	if got != sentinel {
		t.Fatalf("MgmntClientFor without creds should return SDKInstance singleton; got %p want %p", got, sentinel)
	}
}

func TestMgmntClientFor_WithCredentials_CachesByKey(t *testing.T) {
	resetSDKGlobals(t)
	SDKInstance = mgmnt.NewClientWithUrls("singleton", "https://hub.example.com", "https://mgmnt.example.com/", false)

	creds := tenant.Credentials{
		ServiceToken: "tenant-tok",
		HubBaseURL:   "https://hub.tenant.example.com",
		MgmntBaseURL: "https://mgmnt.tenant.example.com",
	}
	ctx := tenant.WithCredentials(context.Background(), creds)

	first := MgmntClientFor(ctx)
	if first == nil {
		t.Fatalf("expected a client, got nil")
	}
	if first == SDKInstance {
		t.Fatalf("expected a tenant-scoped client distinct from singleton; got singleton")
	}

	second := MgmntClientFor(ctx)
	if first != second {
		t.Fatalf("expected cached client on second call; got %p then %p", first, second)
	}

	// A different tenant key must produce a different client.
	otherCtx := tenant.WithCredentials(context.Background(), tenant.Credentials{
		ServiceToken: "other-tok",
		HubBaseURL:   creds.HubBaseURL,
		MgmntBaseURL: creds.MgmntBaseURL,
	})
	other := MgmntClientFor(otherCtx)
	if other == first {
		t.Fatalf("expected distinct client for different service token, got same %p", first)
	}
}

func TestAuthExpiryTransport_401Calls_MarkSessionDead(t *testing.T) {
	resetSDKGlobals(t)

	var hits int32
	MarkSessionDead = func(ctx context.Context) {
		atomic.AddInt32(&hits, 1)
	}

	rt := &authExpiryTransport{
		Base: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusUnauthorized, Body: http.NoBody}, nil
		}),
	}

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.com/", nil)
	if _, err := rt.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip returned error: %v", err)
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Fatalf("expected MarkSessionDead to fire once on 401, got %d", got)
	}
}

func TestAuthExpiryTransport_Non401DoesNotCall_MarkSessionDead(t *testing.T) {
	resetSDKGlobals(t)

	var hits int32
	MarkSessionDead = func(ctx context.Context) {
		atomic.AddInt32(&hits, 1)
	}

	rt := &authExpiryTransport{
		Base: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
		}),
	}

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.com/", nil)
	if _, err := rt.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip returned error: %v", err)
	}
	if got := atomic.LoadInt32(&hits); got != 0 {
		t.Fatalf("expected MarkSessionDead NOT to fire on 200, got %d", got)
	}
}

func TestAuthExpiryTransport_NilMarkSessionDead_DoesNotPanic(t *testing.T) {
	resetSDKGlobals(t)
	MarkSessionDead = nil

	rt := &authExpiryTransport{
		Base: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusUnauthorized, Body: http.NoBody}, nil
		}),
	}

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.com/", nil)
	if _, err := rt.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip returned error: %v", err)
	}
	// reaching here without panic is the assertion
}

func TestAuthExpiryTransport_NilBase_FallsBackToDefaultTransport(t *testing.T) {
	resetSDKGlobals(t)
	rt := &authExpiryTransport{Base: nil}
	// We don't make a real network call; just confirm that constructing and
	// invoking RoundTrip with a Nil-base does not panic. Use a clearly bogus
	// URL so http.DefaultTransport fails fast instead of dialing the network.
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://127.0.0.1:0/", nil)
	_, _ = rt.RoundTrip(req)
}

func TestGetSuprSendWorkspaceClient_VariadicNoCtx_NilSDK_ReturnsError(t *testing.T) {
	resetSDKGlobals(t)

	// With no ctx and a nil SDKInstance there is no client to route through.
	// The nil-guard must return a clean error rather than panicking on a
	// nil-pointer deref.
	SDKInstance = nil
	_, err := GetSuprSendWorkspaceClient("ws")
	if err == nil {
		t.Fatalf("expected an error when SDKInstance is nil and no ctx is supplied, got nil")
	}
}

func TestGetSuprSendWorkspaceClient_VariadicWithCtx_UsesCtxClient(t *testing.T) {
	resetSDKGlobals(t)

	// SDKInstance is deliberately nil — if the call routed through it the
	// test would panic. Routing through ctx should pick the tenant client
	// from MgmntClientFor's cache and only fail later inside
	// GetWorkspaceClient (which we don't drive to completion because the
	// hub URL is unreachable).
	SDKInstance = nil

	creds := tenant.Credentials{
		ServiceToken: "tenant-tok",
		HubBaseURL:   "http://127.0.0.1:0",
		MgmntBaseURL: "http://127.0.0.1:0",
	}
	ctx := tenant.WithCredentials(context.Background(), creds)

	// Call must not panic — the ctx-derived client is non-nil even though
	// SDKInstance is nil. The workspace key/secret lookup will fail with
	// an error; we only assert that we got an error (not a panic) and that
	// no panic surfaced.
	_, err := GetSuprSendWorkspaceClient("ws", ctx)
	if err == nil {
		t.Fatalf("expected workspace key/secret lookup error (unreachable URL), got nil")
	}
}

// TestGetWorkspaceClientCtx_ConcurrentAccess exercises the mutex added in
// mgmnt/client.go: many goroutines hitting GetWorkspaceClientCtx on one shared
// *SS_MgmntClient. A local httptest server returns a valid key/secret so the
// cold path actually creates a workspace client and WRITES the
// workspaceClients map — the line that races without the mutex. Run with
// -race to catch a regression of the data race.
func TestGetWorkspaceClientCtx_ConcurrentAccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"key":"k","secret":"s"}`))
	}))
	defer srv.Close()

	// hub URL points at the test server; mgmnt URL is unused on this path.
	c := mgmnt.NewClientWithUrls("tok", srv.URL, srv.URL, false)

	const n = 32
	var wg sync.WaitGroup
	wg.Add(n)
	for i := range n {
		go func(i int) {
			defer wg.Done()
			// Mix of workspaces to stir both the create and read-cached paths.
			ws := fmt.Sprintf("ws-%d", i%4)
			if _, err := c.GetWorkspaceClientCtx(context.Background(), ws); err != nil {
				t.Errorf("GetWorkspaceClientCtx(%s): %v", ws, err)
			}
		}(i)
	}
	wg.Wait()
}
