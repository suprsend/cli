package event

import (
	"errors"
	"strings"
	"testing"

	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/mgmnt"
)

func TestPushFailuresNamesRejectedEvents(t *testing.T) {
	events := []any{
		map[string]any{"name": "signed_up"},
		map[string]any{"name": "order_placed"},
	}
	records := []mgmnt.EventPushRecord{
		{StatusCode: 201, Status: "success"},
		{StatusCode: 400, Status: "error", Message: "Schema 'order#3' not found or is not active."},
	}
	err := PushFailures(events, records)
	var ce *clierr.CLIError
	if !errors.As(err, &ce) || ce.Code != clierr.CodeAPIInternal {
		t.Fatalf("expected a clierr with code %q, got %#v", clierr.CodeAPIInternal, err)
	}
	if !strings.Contains(ce.Message, "1 of 2 event(s)") || !strings.Contains(ce.Message, "events/order_placed: Schema 'order#3'") {
		t.Errorf("unexpected message: %s", ce.Message)
	}
	if strings.Contains(ce.Message, "signed_up") {
		t.Errorf("a successful event is listed as failed: %s", ce.Message)
	}

	if err := PushFailures(events, records[:1]); err != nil {
		t.Errorf("expected no error when every record succeeded, got %v", err)
	}
}
