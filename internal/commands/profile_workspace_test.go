package commands

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

// profile add/modify's --workspace is the value stored in the profile. With
// an active profile whose token resolves, the root hook used to resolve it
// like a target workspace: add copied the active profile's workspace into the
// new profile, and modify --workspace "" failed as an invalid workspace.
func TestProfileWorkspaceFlagIsNotResolved(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("SUPRSEND_SERVICE_TOKEN", "")
	t.Setenv("SUPRSEND_WORKSPACE", "")
	path := filepath.Join(t.TempDir(), "suprsend.yaml")
	if err := os.WriteFile(path, []byte(`active_profile: p1
profiles:
  p1:
    base_url: https://hub.suprsend.com/
    mgmnt_url: https://management-api.suprsend.com/
    service_token: tok1
    workspace: production
`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	run := func(args ...string) {
		t.Helper()
		rootCmd.SetArgs(append(args, "--config", path))
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	workspaces := func() map[string]any {
		t.Helper()
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var cfg struct {
			Profiles map[string]map[string]any `yaml:"profiles"`
		}
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			t.Fatal(err)
		}
		ws := map[string]any{}
		for name, p := range cfg.Profiles {
			ws[name] = p["workspace"]
		}
		return ws
	}

	run("profile", "add", "--name", "p2", "--service-token", "tok2")
	if ws := workspaces()["p2"]; ws != nil {
		t.Errorf("profile add without --workspace stored workspace %v, want none", ws)
	}

	run("profile", "modify", "--name", "p1", "--workspace", "")
	if ws := workspaces()["p1"]; ws != nil {
		t.Errorf(`profile modify --workspace "" left workspace %v, want it removed`, ws)
	}
}
