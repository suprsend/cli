package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func createTempConfig(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".suprsend.yaml")
	// Create a minimal valid config
	content := "active_profile: \"\"\nprofiles: {}\n"
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create temp config: %v", err)
	}
	return configPath
}

func TestProfileAdd(t *testing.T) {
	configPath := createTempConfig(t)
	_, _, exitCode := runCLI(t,
		"--config", configPath,
		"profile", "add",
		"--name", "test-profile",
		"--service-token", "test-token-12345",
	)
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
	// Verify config file contains the profile
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	if !strings.Contains(string(data), "test-profile") {
		t.Errorf("config should contain 'test-profile', got: %s", string(data))
	}
}

func TestProfileList(t *testing.T) {
	configPath := createTempConfig(t)
	// Add a profile first
	runCLI(t, "--config", configPath, "profile", "add", "--name", "prof1", "--service-token", "tok1")
	runCLI(t, "--config", configPath, "profile", "add", "--name", "prof2", "--service-token", "tok2")

	stdout, _, exitCode := runCLI(t, "--config", configPath, "profile", "list")
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
	if !strings.Contains(stdout, "prof1") {
		t.Errorf("output should contain 'prof1', got: %s", stdout)
	}
	if !strings.Contains(stdout, "prof2") {
		t.Errorf("output should contain 'prof2', got: %s", stdout)
	}
}

func TestProfileUse(t *testing.T) {
	configPath := createTempConfig(t)
	// Add profiles
	runCLI(t, "--config", configPath, "profile", "add", "--name", "prof1", "--service-token", "tok1")
	runCLI(t, "--config", configPath, "profile", "add", "--name", "prof2", "--service-token", "tok2")

	// Switch to prof2
	_, _, exitCode := runCLI(t, "--config", configPath, "profile", "use", "--name", "prof2")
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}

	// Verify active profile changed
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	if !strings.Contains(string(data), "active_profile: prof2") {
		t.Errorf("config should have active_profile: prof2, got: %s", string(data))
	}
}

func TestProfileRemove(t *testing.T) {
	configPath := createTempConfig(t)
	// Add a profile
	runCLI(t, "--config", configPath, "profile", "add", "--name", "to-remove", "--service-token", "tok1")

	// Verify it exists
	data, _ := os.ReadFile(configPath)
	if !strings.Contains(string(data), "to-remove") {
		t.Fatal("profile should exist before removal")
	}

	// Remove it
	_, _, exitCode := runCLI(t, "--config", configPath, "profile", "remove", "--name", "to-remove")
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}

	// Verify it's gone
	data, _ = os.ReadFile(configPath)
	if strings.Contains(string(data), "to-remove") {
		t.Errorf("profile should be removed, got: %s", string(data))
	}
}
