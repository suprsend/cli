package tenant_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/suprsend/cli/pkg/tenant"
)

func TestContextRoundTrip(t *testing.T) {
	ctx := context.Background()

	if _, err := tenant.FromContext(ctx); !errors.Is(err, tenant.ErrNoCredentials) {
		t.Fatalf("empty ctx: got err = %v, want ErrNoCredentials", err)
	}

	creds := tenant.Credentials{
		ServiceToken: "tok-123",
		Workspace:    "staging",
	}
	ctx = tenant.WithCredentials(ctx, creds)

	got, err := tenant.FromContext(ctx)
	if err != nil {
		t.Fatalf("FromContext: %v", err)
	}
	if got != creds {
		t.Errorf("round-trip mismatch: got %+v, want %+v", got, creds)
	}
}

func TestCredentialsStringRedactsToken(t *testing.T) {
	creds := tenant.Credentials{ServiceToken: "super-secret-token-value-xyz", Workspace: "staging"}
	for _, verb := range []string{"%v", "%+v", "%s"} {
		out := fmt.Sprintf(verb, creds)
		if strings.Contains(out, "super-secret-token-value-xyz") {
			t.Errorf("fmt %q leaked token: %s", verb, out)
		}
		if !strings.Contains(out, "REDACTED") {
			t.Errorf("fmt %q did not include REDACTED marker: %s", verb, out)
		}
	}
}

func TestCredentialsJSONRedactsToken(t *testing.T) {
	creds := tenant.Credentials{ServiceToken: "super-secret-token-value-xyz", Workspace: "staging"}
	payload, err := json.Marshal(creds)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if strings.Contains(string(payload), "super-secret-token-value-xyz") {
		t.Errorf("json.Marshal leaked token: %s", payload)
	}
	if !strings.Contains(string(payload), "[REDACTED]") {
		t.Errorf("json.Marshal did not include [REDACTED] marker: %s", payload)
	}
}
