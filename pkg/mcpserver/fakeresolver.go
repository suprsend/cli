package mcpserver

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

// FakeResolver is an in-memory TenantResolver useful for tests and for
// running the library standalone without wiring real auth. Tenants are keyed
// by bearer token in the Authorization header.
type FakeResolver struct {
	mu      sync.RWMutex
	tenants map[string]*Tenant
}

// NewFakeResolver returns an empty FakeResolver. Add tenants with Add.
func NewFakeResolver() *FakeResolver {
	return &FakeResolver{tenants: map[string]*Tenant{}}
}

// Add registers a tenant under the given bearer token.
func (f *FakeResolver) Add(token string, t *Tenant) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tenants[token] = t
}

// Resolve implements TenantResolver. Returns ErrUnauthorized if the request
// has no Authorization header or the bearer token is unknown.
func (f *FakeResolver) Resolve(_ context.Context, r *http.Request) (*Tenant, error) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return nil, fmt.Errorf("missing Authorization header: %w", ErrUnauthorized)
	}
	token := strings.TrimPrefix(auth, "Bearer ")
	f.mu.RLock()
	defer f.mu.RUnlock()
	t, ok := f.tenants[token]
	if !ok {
		return nil, fmt.Errorf("unknown tenant: %w", ErrUnauthorized)
	}
	return t, nil
}
