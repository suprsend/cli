package cli_test

import (
	"testing"
)

func TestVersionCommand_ExitCode(t *testing.T) {
	_, _, exitCode := runCLI(t, "version")
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
}

func TestVersionCommand_Output(t *testing.T) {
	stdout, _, _ := runCLI(t, "version")
	if stdout == "" {
		t.Error("expected version output, got empty string")
	}
}
