package event

import (
	"errors"
	"testing"

	"github.com/spf13/pflag"
	"github.com/suprsend/cli/internal/clierr"
)

// A --json payload without a non-empty "events" array is a mistake in the
// payload, not "nothing to push", so it must fail without calling the API.
func TestEventPushJSONRequiresEventsArray(t *testing.T) {
	prevSilenceErrors, prevSilenceUsage := EventCmd.SilenceErrors, EventCmd.SilenceUsage
	EventCmd.SilenceErrors, EventCmd.SilenceUsage = true, true
	t.Cleanup(func() {
		EventCmd.SilenceErrors, EventCmd.SilenceUsage = prevSilenceErrors, prevSilenceUsage
		EventCmd.SetArgs(nil)
		eventPushCmd.Flags().VisitAll(func(f *pflag.Flag) {
			_ = f.Value.Set(f.DefValue)
			f.Changed = false
		})
	})

	for _, payload := range []string{`{}`, `{"event":[{"name":"x"}]}`, `{"events":[]}`, `{"events":"x"}`} {
		EventCmd.SetArgs([]string{"push", "--dry-run", "--json", payload})
		err := EventCmd.Execute()
		var ce *clierr.CLIError
		if !errors.As(err, &ce) || ce.Code != clierr.CodeInvalidUsage {
			t.Errorf("payload %s: expected a clierr with code %q, got %#v", payload, clierr.CodeInvalidUsage, err)
		}
	}
}
