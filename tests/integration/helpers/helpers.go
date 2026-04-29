package helpers

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// AssertExitCode asserts that err is an *exec.ExitError with the given exit code.
func AssertExitCode(t *testing.T, err error, code int) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected exit code %d, but command exited 0", code)
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected *exec.ExitError, got %T: %v", err, err)
	}
	if exitErr.ExitCode() != code {
		t.Errorf("expected exit code %d, got %d", code, exitErr.ExitCode())
	}
}

const BinaryName = "suprsend-cov"

func BinaryPath(t *testing.T) string {
	t.Helper()
	root := RepoRoot(t)
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	return filepath.Join(root, BinaryName+ext)
}

func RepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// helpers/ is three levels below repo root: tests/integration/helpers/helpers.go
	return filepath.Join(filepath.Dir(file), "..", "..", "..")
}

// TestdataPath returns the absolute path to a file under tests/integration/testdata/.
func TestdataPath(t *testing.T, parts ...string) string {
	t.Helper()
	root := RepoRoot(t)
	return filepath.Join(append([]string{root, "tests", "integration", "testdata"}, parts...)...)
}

// RunCLI executes the instrumented binary with the given args and extra env vars.
// If GOCOVERDIR is already set in the environment, that directory is used so coverage
// accumulates across the full test run.
func RunCLI(t *testing.T, env []string, args ...string) (string, error) {
	t.Helper()
	covDir := os.Getenv("GOCOVERDIR")
	if covDir == "" {
		covDir = t.TempDir()
	}
	cmd := exec.Command(BinaryPath(t), args...)
	cmd.Env = append(os.Environ(), append(env, "GOCOVERDIR="+covDir)...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
