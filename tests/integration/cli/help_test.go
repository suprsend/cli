package cli_test

import (
	"strings"
	"testing"
)

func TestRootHelp(t *testing.T) {
	stdout, _, exitCode := runCLI(t, "--help")
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
	if !strings.Contains(strings.ToLower(stdout), "suprsend") {
		t.Errorf("help output should contain 'suprsend', got: %s", stdout)
	}
}

func TestWorkflowHelp(t *testing.T) {
	stdout, _, exitCode := runCLI(t, "workflow", "--help")
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
	if stdout == "" {
		t.Error("expected help output, got empty string")
	}
}

func TestSchemaHelp(t *testing.T) {
	stdout, _, exitCode := runCLI(t, "schema", "--help")
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
	if stdout == "" {
		t.Error("expected help output, got empty string")
	}
}

func TestUnknownCommand(t *testing.T) {
	_, _, exitCode := runCLI(t, "nonexistent-command-xyz")
	if exitCode == 0 {
		t.Error("expected non-zero exit code for unknown command")
	}
}

func TestNoArgs(t *testing.T) {
	stdout, _, _ := runCLI(t, )
	// With no args, should show help or usage
	if !strings.Contains(strings.ToLower(stdout), "suprsend") && stdout != "" {
		t.Errorf("expected help-like output, got: %s", stdout)
	}
}
