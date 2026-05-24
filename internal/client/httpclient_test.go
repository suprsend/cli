package client

import (
	"net/http"
	"testing"
	"time"
)

// roundTripperFunc lets a test assert the injected transport is honoured.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestNewHTTPClientWithOptions_WiresTransportAndTimeout(t *testing.T) {
	var calls int
	rt := roundTripperFunc(func(*http.Request) (*http.Response, error) {
		calls++
		return nil, http.ErrUseLastResponse
	})

	rc := NewHTTPClientWithOptions(Options{
		Transport: rt,
		Timeout:   10 * time.Second,
	})
	defer rc.Close()

	if got := rc.Timeout(); got != 10*time.Second {
		t.Fatalf("expected 10s timeout, got %v", got)
	}

	// Confirm Transport is the one we passed in (round-trips through it).
	if _, err := rc.Transport().RoundTrip(&http.Request{}); err == nil && calls == 0 {
		t.Fatalf("expected injected RoundTripper to be invoked")
	}
	if calls != 1 {
		t.Fatalf("expected injected RoundTripper to be invoked once, got %d", calls)
	}
}

func TestNewHTTPClientWithOptions_NoOptionsLeavesDefaults(t *testing.T) {
	rc := NewHTTPClientWithOptions(Options{})
	defer rc.Close()

	if got := rc.Timeout(); got != 0 {
		t.Fatalf("expected zero timeout when not set, got %v", got)
	}
}

func TestNewHTTPClient_LegacyMatchesEmptyOptions(t *testing.T) {
	rc := NewHTTPClient()
	defer rc.Close()

	if got := rc.Timeout(); got != 0 {
		t.Fatalf("expected zero timeout from legacy ctor, got %v", got)
	}
}
