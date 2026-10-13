package clierr

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFailuresListsEachFailure(t *testing.T) {
	err := Failures("2 workflow(s) failed to push", []string{"workflows/a: boom", "workflows/b: invalid"}, CodeAPIInternal)
	if want := "2 workflow(s) failed to push:\nworkflows/a: boom\nworkflows/b: invalid"; err.Message != want {
		t.Errorf("Message = %q, want %q", err.Message, want)
	}
	if err.ExitCode() != ExitGeneralError {
		t.Errorf("ExitCode = %d, want %d", err.ExitCode(), ExitGeneralError)
	}
	var out struct {
		Details struct {
			Failures []string `json:"failures"`
		} `json:"details"`
	}
	if jerr := json.Unmarshal(err.JSON(), &out); jerr != nil || len(out.Details.Failures) != 2 {
		t.Errorf("details.failures = %v (%v), want both failures", out.Details.Failures, jerr)
	}
	if strings.Contains(Failures("x", nil, CodeAPIInternal).Message, ":") {
		t.Error("no failures should leave the summary unchanged")
	}
}
