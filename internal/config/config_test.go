package config

import (
	"errors"
	"os"
	"testing"

	"github.com/suprsend/cli/internal/clierr"
)

func TestResolveWorkspace(t *testing.T) {
	tests := []struct {
		name       string
		flag       string
		flagSet    bool
		env        string
		profile    string
		want       string
		wantSource ConfigSource
	}{
		{"default", "", false, "", "", DefaultWorkspace, ConfigSourceDefault},
		{"profile", "", false, "", "evals-pool-1", "evals-pool-1", ConfigSourceProfile},
		{"env beats profile", "", false, "evals-staging", "evals-pool-1", "evals-staging", ConfigSourceEnv},
		{"flag beats env", "production", true, "evals-staging", "evals-pool-1", "production", ConfigSourceFlag},
		// A shell quoting slip ("-w evals-staging" as one argument) gives a leading space.
		{"flag is trimmed", " evals-staging", true, "", "", "evals-staging", ConfigSourceFlag},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SUPRSEND_WORKSPACE", tt.env)
			c := &Config{profileWorkspace: tt.profile}
			got, err := c.ResolveWorkspace(tt.flag, tt.flagSet)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Value != tt.want || got.Source != tt.wantSource {
				t.Errorf("got %q from %s, want %q from %s", got.Value, got.Source, tt.want, tt.wantSource)
			}
			if c.Workspace != got {
				t.Errorf("Cfg.Workspace = %+v, want %+v", c.Workspace, got)
			}
		})
	}
}

func TestResolveWorkspaceRejectsNonSlugs(t *testing.T) {
	for _, bad := range []string{"", "  ", "evals staging", "ws/../x", "-w"} {
		_, err := (&Config{}).ResolveWorkspace(bad, true)
		var ce *clierr.CLIError
		if !errors.As(err, &ce) || ce.Code != clierr.CodeInvalidUsage {
			t.Errorf("workspace %q: got %v, want an invalid-usage error", bad, err)
		}
	}
}

// An explicit --service-token wins over SUPRSEND_SERVICE_TOKEN, which wins
// over the profile.
func TestServiceTokenPrecedence(t *testing.T) {
	profile := Profile{ServiceToken: ConfigString{Value: "from-profile"}}

	t.Setenv("SUPRSEND_SERVICE_TOKEN", "from-env")
	if got, _ := GetResolvedServiceToken("from-flag", profile); got.Value != "from-flag" || got.Source != ConfigSourceFlag {
		t.Errorf("flag and env set: got %q from %s, want the flag", got.Value, got.Source)
	}
	if got, _ := GetResolvedServiceToken("", profile); got.Value != "from-env" || got.Source != ConfigSourceEnv {
		t.Errorf("env set: got %q from %s, want the env var", got.Value, got.Source)
	}

	t.Setenv("SUPRSEND_SERVICE_TOKEN", "")
	if got, _ := GetResolvedServiceToken("", profile); got.Value != "from-profile" || got.Source != ConfigSourceProfile {
		t.Errorf("only profile set: got %q from %s, want the profile", got.Value, got.Source)
	}
	if _, err := GetResolvedServiceToken("", Profile{}); err == nil {
		t.Error("no token anywhere: expected an error")
	}
}

// --profile beats SUPRSEND_PROFILE, which beats the config file's active
// profile; a name that isn't in the file is an error, not a silent fallback.
func TestResolveProfile(t *testing.T) {
	path := t.TempDir() + "/suprsend.yaml"
	cfgFile := `active_profile: a
profiles:
  a:
    mgmnt_url: https://a.example.test/
    service_token: tok-a
  b:
    mgmnt_url: https://b.example.test/
    service_token: tok-b
    workspace: dev
`
	if err := os.WriteFile(path, []byte(cfgFile), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SUPRSEND_SERVICE_TOKEN", "")
	t.Setenv("SUPRSEND_MGMNT_URL", "")
	t.Setenv("SUPRSEND_WORKSPACE", "")
	prev := *Cfg
	t.Cleanup(func() { *Cfg = prev })

	tests := []struct {
		name, flag, env, want string
		source                ConfigSource
		host                  string
	}{
		{"active profile", "", "", "a", ConfigSourceActive, "https://a.example.test/"},
		{"env", "", "b", "b", ConfigSourceEnv, "https://b.example.test/"},
		{"flag beats env", "a", "b", "a", ConfigSourceFlag, "https://a.example.test/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SUPRSEND_PROFILE", tt.env)
			if err := Cfg.Resolve(FlagValues{CfgFile: path, Profile: tt.flag}); err != nil {
				t.Fatal(err)
			}
			if Cfg.Profile.Value != tt.want || Cfg.Profile.Source != tt.source || Cfg.MgmntUrl.Value != tt.host {
				t.Errorf("got profile %q from %s on %s, want %q from %s on %s",
					Cfg.Profile.Value, Cfg.Profile.Source, Cfg.MgmntUrl.Value, tt.want, tt.source, tt.host)
			}
		})
	}

	t.Run("profile workspace follows the chosen profile", func(t *testing.T) {
		t.Setenv("SUPRSEND_PROFILE", "b")
		if err := Cfg.Resolve(FlagValues{CfgFile: path}); err != nil {
			t.Fatal(err)
		}
		if ws, _ := Cfg.ResolveWorkspace("", false); ws.Value != "dev" {
			t.Errorf("workspace = %q, want dev from profile b", ws.Value)
		}
	})

	t.Run("unknown profile", func(t *testing.T) {
		t.Setenv("SUPRSEND_PROFILE", "")
		err := Cfg.Resolve(FlagValues{CfgFile: path, Profile: "nope"})
		var ce *clierr.CLIError
		if !errors.As(err, &ce) || ce.Code != clierr.CodeConfigInvalid {
			t.Errorf("got %v, want a config-invalid error", err)
		}
	})
}
