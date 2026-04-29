package helpers

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/creack/pty"
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
	return RunCLIInDir(t, "", env, args...)
}

// RunCLIInDir is like RunCLI but sets the working directory of the subprocess.
// Pass an empty string to inherit the test binary's working directory.
func RunCLIInDir(t *testing.T, dir string, env []string, args ...string) (string, error) {
	t.Helper()
	return RunCLIWithStdin(t, dir, nil, env, args...)
}

// RunCLIWithStdin is like RunCLIInDir but also pipes the given reader to the subprocess stdin.
// Pass nil for stdin to use the default (os.DevNull).
func RunCLIWithStdin(t *testing.T, dir string, stdin io.Reader, env []string, args ...string) (string, error) {
	t.Helper()
	covDir := os.Getenv("GOCOVERDIR")
	if covDir == "" {
		covDir = t.TempDir()
	}
	cmd := exec.Command(BinaryPath(t), args...)
	cmd.Dir = dir
	cmd.Stdin = stdin
	cmd.Env = append(os.Environ(), append(env, "GOCOVERDIR="+covDir)...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// RunCLIWithPTY starts the binary under a PTY so IsInputInteractive() returns true.
// input is written to the PTY master after a brief delay to let the prompt appear.
// Returns combined output collected from the PTY master.
func RunCLIWithPTY(t *testing.T, dir string, input string, env []string, args ...string) (string, error) {
	t.Helper()
	covDir := os.Getenv("GOCOVERDIR")
	if covDir == "" {
		covDir = t.TempDir()
	}
	cmd := exec.Command(BinaryPath(t), args...)
	cmd.Dir = dir
	// TERM=dumb prevents termenv from sending cursor-position queries (ESC[6n)
	// and waiting for terminal responses, which would block the PTY indefinitely.
	cmd.Env = append(os.Environ(), append(env, "GOCOVERDIR="+covDir, "TERM=dumb")...)

	ptm, err := pty.Start(cmd)
	if err != nil {
		return "", err
	}

	// Drain the PTY master in a goroutine; on macOS the master returns EIO once
	// the slave is closed, so we rely on cmd.Wait() + ptm.Close() to unblock it.
	var buf bytes.Buffer
	copyDone := make(chan struct{})
	go func() {
		io.Copy(&buf, ptm)
		close(copyDone)
	}()

	// Give the process time to start and display the prompt, then send input.
	time.Sleep(150 * time.Millisecond)
	io.WriteString(ptm, input)

	// Wait for the process to finish with a timeout.
	waitDone := make(chan error, 1)
	go func() { waitDone <- cmd.Wait() }()

	var waitErr error
	select {
	case waitErr = <-waitDone:
	case <-time.After(10 * time.Second):
		t.Logf("subprocess timed out; output so far:\n%s", buf.String())
		cmd.Process.Kill()
		waitErr = <-waitDone
	}

	ptm.Close()
	select {
	case <-copyDone:
	case <-time.After(2 * time.Second):
	}

	return buf.String(), waitErr
}
