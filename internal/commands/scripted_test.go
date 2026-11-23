package commands

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/suprsend/cli/internal/clierr"
	"gopkg.in/yaml.v3"
)

// With --name and --service-token, profile add works without a terminal
// even when there is no config file yet, and the first profile is made the
// active one.
func TestProfileAddCreatesConfigFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("SUPRSEND_SERVICE_TOKEN", "")
	t.Setenv("SUPRSEND_PROFILE", "")
	path := filepath.Join(t.TempDir(), "suprsend.yaml")
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	rootCmd.SetArgs([]string{"profile", "add", "--name", "ci", "--service-token", "tok-ci", "--config", path})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("config file not created: %v", err)
	}
	var cfg struct {
		Active   string                    `yaml:"active_profile"`
		Profiles map[string]map[string]any `yaml:"profiles"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Active != "ci" || cfg.Profiles["ci"]["service_token"] != "tok-ci" {
		t.Errorf("unexpected config:\n%s", data)
	}
	if info, _ := os.Stat(path); info.Mode().Perm() != 0o600 {
		t.Errorf("config file mode = %v, want 0600 (it holds tokens)", info.Mode().Perm())
	}
}

// SUPRSEND_REQUIRE_WORKSPACE turns a write that would fall back to the
// default workspace into an error, before anything is sent.
func TestRequireWorkspace(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("SUPRSEND_SERVICE_TOKEN", "tok")
	t.Setenv("SUPRSEND_PROFILE", "")
	t.Setenv("SUPRSEND_WORKSPACE", "")
	t.Setenv("SUPRSEND_REQUIRE_WORKSPACE", "1")
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	// --config is reset: cobra keeps flag values from earlier tests.
	rootCmd.SetArgs([]string{"workflow", "push", "--dry-run", "--dir", t.TempDir(), "--config", ""})
	err := rootCmd.Execute()
	var ce *clierr.CLIError
	if !errors.As(err, &ce) || ce.Code != clierr.CodeInvalidUsage {
		t.Errorf("got %v, want an invalid-usage error", err)
	}
}
