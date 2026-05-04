package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/config"
	"github.com/suprsend/cli/internal/utils"
)

type EnvVar struct {
	Name        string `json:"name"`
	Value       string `json:"value"`
	Source      string `json:"source"`
	Description string `json:"description"`
}

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Show recognized environment variables and their current values",
	Long: `Print all environment variables recognized by suprsend, their current values,
and how each one affects the CLI. Values for sensitive variables (tokens) are
redacted. Useful for verifying configuration in CI/CD pipelines and agent contexts.`,
	Example: `  suprsend env`,
	RunE: func(cmd *cobra.Command, args []string) error {
		outputType, _ := cmd.Flags().GetString("output")
		if err := utils.ValidateOutputType(outputType, "pretty", "json", "yaml"); err != nil {
			return err
		}
		utils.OutputData(buildEnvRows(), outputType)
		return nil
	},
}

func buildEnvRows() []EnvVar {
	cfg := config.Cfg
	return []EnvVar{
		{
			Name:        "SUPRSEND_SERVICE_TOKEN",
			Value:       serviceTokenDisplay(cfg.ServiceToken),
			Source:      string(cfg.ServiceToken.Source),
			Description: "Service token for authentication (overrides --service-token flag and config file)",
		},
		{
			Name:        "SUPRSEND_BASE_URL",
			Value:       cfg.BaseUrl.Value,
			Source:      string(cfg.BaseUrl.Source),
			Description: "Base API URL for BYOC/self-hosted instances (overrides default)",
		},
		{
			Name:        "SUPRSEND_MGMNT_URL",
			Value:       cfg.MgmntUrl.Value,
			Source:      string(cfg.MgmntUrl.Source),
			Description: "Management API URL for BYOC/self-hosted instances (overrides default)",
		},
		{
			Name:        "NO_COLOR",
			Value:       fmt.Sprintf("%v", cfg.NoColorOutput.Value),
			Source:      string(cfg.NoColorOutput.Source),
			Description: "Disable color output when set to any non-empty value",
		},
		{
			Name:        "HTTP_PROXY",
			Value:       proxyDisplay(cfg.ProxyURL),
			Source:      string(cfg.ProxyURL.Source),
			Description: "HTTP proxy URL for outbound requests",
		},
	}
}

func serviceTokenDisplay(tok config.ConfigString) string {
	if tok.Value == "" {
		return "(unset)"
	}
	return fmt.Sprintf("<redacted, %d chars>", len(tok.Value))
}


func proxyDisplay(proxy config.ConfigString) string {
	if proxy.Value == "" {
		return "(unset)"
	}
	return proxy.Value
}

func init() {
	rootCmd.AddCommand(envCmd)
	envCmd.Flags().StringP("output", "o", "pretty", "Output format: pretty, json, or yaml")
}
