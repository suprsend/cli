package translation

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/utils"
)

var translationListCmd = &cobra.Command{
	Use:   "list",
	Short: "List preference translations",
	Long:  "List available translation locales for preference categories in a workspace. Returns the locale codes that have translations configured.",
	Example: `  # List available translation locales
  suprsend category translation list

  # List with JSON output
  suprsend category translation list --output json

  # List in the production workspace
  suprsend category translation list --workspace production`,
	Annotations: map[string]string{
		"skills:tip:output": "Use `-o json` for machine-readable JSON output, `-o yaml` for YAML. Default `-o pretty` outputs a human-friendly table.",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		outputType, _ := cmd.Flags().GetString("output")

		return listTranslations(cmd.Context(), workspace, outputType)
	},
}

func listTranslations(ctx context.Context, workspace, outputType string) error {
	if workspace == "" {
		return clierr.New("workspace flag is required", clierr.CodeInvalidUsage)
	}

	mgmntClient := utils.GetSuprSendMgmntClient()

	spinner := utils.NewSpinner("Loading...")

	translations, err := mgmntClient.ListPreferenceTranslations(ctx, workspace)
	if err != nil {
		return fmt.Errorf("couldn't fetch translations: %w", err)
	}

	spinner.Stop(fmt.Sprintf("Listed %d translation locales from %s", len(translations.Results), workspace))

	if len(translations.Results) == 0 && utils.IsOutputPiped() {
		utils.OutputData([]any{}, outputType)
		return nil
	}
	utils.OutputData(translations.Results, outputType)
	return nil
}

func init() {
	TranslationCmd.AddCommand(translationListCmd)
}
