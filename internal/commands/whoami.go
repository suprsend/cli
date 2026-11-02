package commands

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/config"
	"github.com/suprsend/cli/internal/utils"
)

type whoamiInfo struct {
	Profile            string `json:"profile" yaml:"profile"`
	ConfigFile         string `json:"config_file" yaml:"config_file"`
	Workspace          string `json:"workspace" yaml:"workspace"`
	WorkspaceSource    string `json:"workspace_source" yaml:"workspace_source"`
	MgmntURL           string `json:"mgmnt_url" yaml:"mgmnt_url"`
	BaseURL            string `json:"base_url" yaml:"base_url"`
	ServiceToken       string `json:"service_token" yaml:"service_token"`
	ServiceTokenSource string `json:"service_token_source" yaml:"service_token_source"`
	Checked            bool   `json:"checked" yaml:"checked"`
}

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show which profile, host and workspace commands will use",
	Long: `Print where commands will go: the active profile, the management API host,
the workspace a command would use (and where that value came from: -w,
SUPRSEND_WORKSPACE, the profile, or the built-in default), and where the
service token came from. The token itself is never printed. With --check, it
also calls the API to confirm the token works and the workspace exists.`,
	Example: `  # Show the target of commands run from this shell
  suprsend whoami

  # Show the target for a specific workspace and confirm it exists
  suprsend whoami -w production --check

  # Machine-readable
  suprsend whoami -o json`,
	Annotations: map[string]string{
		"skills:tip.a-first": "Run `suprsend whoami --check -o json` before writing to a workspace: it shows the host, profile and workspace commands will use without reading the config file (which holds tokens), and fails if the token is invalid or the workspace doesn't exist.",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		check, _ := cmd.Flags().GetBool("check")
		f := cmd.Flags().Lookup("workspace")
		ws, err := config.Cfg.ResolveWorkspace(f.Value.String(), f.Changed)
		if err != nil {
			return err
		}
		cfg := config.Cfg
		info := whoamiInfo{
			Profile:            orNone(cfg.Profile.Value),
			ConfigFile:         config.GetConfigFilePath(),
			Workspace:          ws.Value,
			WorkspaceSource:    string(ws.Source),
			MgmntURL:           cfg.MgmntUrl.Value,
			BaseURL:            cfg.BaseUrl.Value,
			ServiceToken:       serviceTokenDisplay(cfg.ServiceToken),
			ServiceTokenSource: orNone(string(cfg.ServiceToken.Source)),
		}
		if check {
			if cfg.ServiceToken.Value == "" {
				return clierr.New("no service token found in command line, environment, or config file", clierr.CodeAuthMissingToken).
					WithHint("set SUPRSEND_SERVICE_TOKEN or run `suprsend profile add`")
			}
			if err := utils.VerifyWorkspace(ws.Value); err != nil {
				return err
			}
			info.Checked = true
		}

		switch cfg.OutputType.Value {
		case "json", "yaml":
			utils.OutputData(info, cfg.OutputType.Value)
			return nil
		}
		fmt.Printf("profile:       %s (%s)\n", info.Profile, info.ConfigFile)
		fmt.Printf("host:          %s\n", hostname(info.MgmntURL))
		fmt.Printf("workspace:     %s (from %s)\n", info.Workspace, info.WorkspaceSource)
		fmt.Printf("service token: %s (from %s)\n", info.ServiceToken, info.ServiceTokenSource)
		if info.Checked {
			fmt.Println("check:         token works and the workspace exists")
		}
		return nil
	},
}

func orNone(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}

func hostname(rawURL string) string {
	if u, err := url.Parse(rawURL); err == nil && u.Host != "" {
		return u.Host
	}
	return rawURL
}

func init() {
	rootCmd.AddCommand(whoamiCmd)
	whoamiCmd.Flags().StringP("workspace", "w", "", config.WorkspaceFlagHelp)
	whoamiCmd.Flags().Bool("check", false, "Also call the API to confirm the token works and the workspace exists")
}
