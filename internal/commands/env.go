package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/utils"
)

type EnvVar struct {
	Name        string `json:"name"`
	Value       string `json:"value"`
	Description string `json:"description"`
}

var recognizedEnvVarNames = []EnvVar{
	{"SUPRSEND_SERVICE_TOKEN", "", "Service token for authentication (overrides --service-token flag and config file)"},
	{"SUPRSEND_BASE_URL", "", "Base API URL for BYOC/self-hosted instances (overrides default)"},
	{"SUPRSEND_MGMNT_URL", "", "Management API URL for BYOC/self-hosted instances (overrides default)"},
	{"NO_COLOR", "", "Disable color output when set to any non-empty value"},
	{"HTTP_PROXY", "", "HTTP proxy URL for outbound requests"},
}

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Show recognized environment variables and their current values",
	Long: `Print all environment variables recognized by suprsend, their current values,
and how each one affects the CLI. Values for sensitive variables (tokens) are
redacted. Useful for verifying configuration in CI/CD pipelines and agent contexts.`,
	Example: `  suprsend env`,
	Run: func(cmd *cobra.Command, args []string) {
		outputType, _ := cmd.Flags().GetString("output")
		if err := utils.ValidateOutputType(outputType, "pretty", "json", "yaml"); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		var rows []EnvVar
		for _, ev := range recognizedEnvVarNames {
			val := os.Getenv(ev.Name)
			display := "(unset)"
			if val != "" {
				if strings.Contains(ev.Name, "TOKEN") || strings.Contains(ev.Name, "SECRET") {
					display = fmt.Sprintf("<redacted, %d chars>", len(val))
				} else {
					display = val
				}
			}
			rows = append(rows, EnvVar{Name: ev.Name, Value: display, Description: ev.Description})
		}
		utils.OutputData(rows, outputType)
	},
}

func init() {
	rootCmd.AddCommand(envCmd)
	envCmd.Flags().StringP("output", "o", "pretty", "Output format: pretty, json, or yaml")
}
