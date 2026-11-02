package config

import (
	"errors"
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
