package mcpserver

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
)

func TestFakeResolver_NoAuthHeader_ReturnsErrUnauthorized(t *testing.T) {
	r := NewFakeResolver()
	req := httptest.NewRequest("GET", "/", nil)

	_, err := r.Resolve(context.Background(), req)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected error to wrap ErrUnauthorized, got: %v", err)
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
