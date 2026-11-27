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

// A pull that writes fewer events than the API reports must fail, so an
// agent never reads a broken pull as an empty workspace.
func TestCheckPulledAll(t *testing.T) {
	if err := checkPulledAll(12, 0); err == nil {
		t.Error("12 reported, 0 written: expected an error")
	}
	if err := checkPulledAll(12, 12); err != nil {
		t.Errorf("12 reported, 12 written: unexpected error %v", err)
	}
	if err := checkPulledAll(0, 0); err != nil {
		t.Errorf("empty workspace: unexpected error %v", err)
	}
}
