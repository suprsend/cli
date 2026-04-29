package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

type envVar struct {
	name        string
	description string
}

var recognizedEnvVars = []envVar{
	{"SUPRSEND_SERVICE_TOKEN", "Service token for authentication (overrides --service-token flag and config file)"},
	{"SUPRSEND_BASE_URL", "Base API URL for BYOC/self-hosted instances (overrides default)"},
	{"SUPRSEND_MGMNT_URL", "Management API URL for BYOC/self-hosted instances (overrides default)"},
	{"NO_COLOR", "Disable color output when set to any non-empty value"},
	{"HTTP_PROXY", "HTTP proxy URL for outbound requests"},
}

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Show recognized environment variables and their current values",
	Long: `Print all environment variables recognized by suprsend, their current values,
and how each one affects the CLI. Values for sensitive variables (tokens) are
redacted. Useful for verifying configuration in CI/CD pipelines and agent contexts.`,
	Example: `  suprsend env`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Recognized environment variables:")
		fmt.Println()
		for _, ev := range recognizedEnvVars {
			val := os.Getenv(ev.name)
			display := "(unset)"
			if val != "" {
				if strings.Contains(ev.name, "TOKEN") || strings.Contains(ev.name, "SECRET") {
					display = fmt.Sprintf("<redacted, %d chars>", len(val))
				} else {
					display = val
				}
			}
			fmt.Printf("  %-30s  %-30s  %s\n", ev.name, display, ev.description)
		}
	},
}

func init() {
	rootCmd.AddCommand(envCmd)
}
