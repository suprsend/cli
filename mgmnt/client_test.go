package mgmnt

import (
	"net/http"
	"testing"
	"time"
)

// roundTripperFunc lets a test assert that the injected transport is honoured.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestNewClientWithUrlsAndTransport_InjectsTransport(t *testing.T) {
	var calls int
	rt := roundTripperFunc(func(*http.Request) (*http.Response, error) {
		calls++
		return nil, http.ErrUseLastResponse
	})

	c := NewClientWithUrlsAndTransport("tok", "https://hub.example.com", "https://mgmnt.example.com/", rt, false)

	hc := c.httpClient()
	if hc.Transport == nil {
		t.Fatalf("expected injected transport, got nil")
	}
	if hc.Timeout != 10*time.Second {
		t.Fatalf("expected 10s timeout, got %v", hc.Timeout)
	}

	// Confirm Transport is actually the one we passed in (round-trips through it).
	_, _ = hc.Transport.RoundTrip(&http.Request{})
	if calls != 1 {
		t.Fatalf("expected injected RoundTripper to be invoked once, got %d", calls)
	}
}

func TestNewClientWithUrls_DefaultsToHTTPDefaultTransport(t *testing.T) {
	c := NewClientWithUrls("tok", "https://hub.example.com", "https://mgmnt.example.com/", false)
	hc := c.httpClient()
	if hc.Transport != http.DefaultTransport {
		t.Fatalf("expected http.DefaultTransport when no transport injected, got %T", hc.Transport)
	}
	if hc.Timeout != 10*time.Second {
		t.Fatalf("expected 10s timeout, got %v", hc.Timeout)
	}
}
