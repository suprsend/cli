package profiles

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/suprsend/cli/internal/config"
)

// answer feeds lines to the prompts through stdin.
func answer(t *testing.T, lines string) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.WriteString(lines); err != nil {
		t.Fatal(err)
	}
	w.Close()
	orig := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = orig; r.Close() })
}

// In the prompted flow, a --workspace given on the command line is used as
// is; it used to be replaced by the profile's current workspace.
func TestModifyInteractiveKeepsWorkspaceFlag(t *testing.T) {
	path := filepath.Join(t.TempDir(), "suprsend.yaml")
	newCfg := func() *config.ProfileConfig {
		return &config.ProfileConfig{ActiveProfile: "p1", Profiles: map[string]config.Profile{
			"p1": {
				BaseUrl:      config.ConfigString{Value: "https://hub.example.test/"},
				MgmntUrl:     config.ConfigString{Value: "https://mgmt.example.test/"},
				ServiceToken: config.ConfigString{Value: "tok1"},
				Workspace:    config.ConfigString{Value: "staging"},
			},
		}}
	}
	reset := func() {
		modifyName, modifyBaseUrl, modifyMgmntUrl, modifyServiceToken, modifyWorkspace = "", "", "", "", ""
	}
	t.Cleanup(reset)
	saved := func() config.Profile {
		t.Helper()
		cfg, err := config.LoadProfileConfig(path)
		if err != nil {
			t.Fatal(err)
		}
		return cfg.Profiles["p1"]
	}

	// --workspace production: Enter keeps the token and both URLs; no workspace prompt.
	reset()
	modifyName, modifyWorkspace = "p1", "production"
	answer(t, "\n\n\n")
	runModifyInteractive(newCfg(), path, true)
	if p := saved(); p.Workspace.Value != "production" || p.ServiceToken.Value != "tok1" || p.MgmntUrl.Value != "https://mgmt.example.test/" {
		t.Errorf("with --workspace production: saved %+v", p)
	}

	// --workspace "" removes it.
	reset()
	modifyName = "p1"
	answer(t, "\n\n\n")
	runModifyInteractive(newCfg(), path, true)
	if p := saved(); p.Workspace.Value != "" {
		t.Errorf(`with --workspace "": saved workspace %q, want none`, p.Workspace.Value)
	}

	// No --workspace: the prompt is shown and Enter keeps the current one.
	reset()
	modifyName = "p1"
	answer(t, "\n\n\n\n")
	runModifyInteractive(newCfg(), path, false)
	if p := saved(); p.Workspace.Value != "staging" {
		t.Errorf("without --workspace: saved workspace %q, want staging", p.Workspace.Value)
	}
}
