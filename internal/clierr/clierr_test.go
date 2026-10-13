package clierr

import (
	"encoding/json"
	"errors"
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

func TestItemFailureDoesNotRepeatResource(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{New("workflows/bad: saved as draft but not committed", CodeAPIConflict), "workflows/bad: failed to push: saved as draft but not committed"},
		{errors.New("request failed: 500"), "workflows/bad: failed to push: request failed: 500"},
		{errors.New("workflows/badge: other"), "workflows/bad: failed to push: workflows/badge: other"},
	}
	for _, tt := range tests {
		if got := ItemFailure("workflows/bad", "failed to push", tt.err); got != tt.want {
			t.Errorf("ItemFailure(%q) = %q, want %q", tt.err, got, tt.want)
		}
	}
}
