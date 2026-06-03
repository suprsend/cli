package mcpserver

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
)

// A request with no Authorization header carries NO credential, so Resolve
// must return (nil, nil) — the signal the auth middleware maps to missing=true
// (RFC 9728 discovery). Returning ErrUnauthorized here would wrongly report
// missing=false ("credential presented but rejected").
func TestFakeResolver_NoAuthHeader_ResolvesNilNil(t *testing.T) {
	r := NewFakeResolver()
	req := httptest.NewRequest("GET", "/", nil)

	tenant, err := r.Resolve(context.Background(), req)
	if err != nil {
		t.Fatalf("missing header should resolve to (nil, nil); got err = %v", err)
	}
	if tenant != nil {
		t.Fatalf("missing header should resolve to a nil tenant; got %+v", tenant)
	}
}

func TestFakeResolver_UnknownToken_ReturnsErrUnauthorized(t *testing.T) {
	r := NewFakeResolver()
	r.Add("good-token", &Tenant{})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer bad-token")

	_, err := r.Resolve(context.Background(), req)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected error to wrap ErrUnauthorized, got: %v", err)
	}
}

func TestFakeResolver_KnownToken_ReturnsTenant(t *testing.T) {
	r := NewFakeResolver()
	want := &Tenant{}
	r.Add("good-token", want)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer good-token")

	got, err := r.Resolve(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Fatalf("expected same *Tenant pointer, got %p want %p", got, want)
	}
}
