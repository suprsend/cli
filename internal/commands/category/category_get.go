package category

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/utils"
)

var categoryGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get categories and translations",
	Long:  "Retrieve preference categories and their translations from a workspace. Returns the full category structure along with translations for all non-English locales. Use --mode to switch between draft and live versions.",
	Example: `  # Get all categories and translations
  suprsend category get

  # Get draft categories
  suprsend category get --mode draft

  # Get with JSON output
  suprsend category get --output json`,
	Annotations: map[string]string{
		"skills:tip:output": "Use `-o json` for machine-readable JSON output, `-o yaml` for YAML.",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		mode, _ := cmd.Flags().GetString("mode")
		outputType, _ := cmd.Flags().GetString("output")
		if err := utils.ValidateOutputType(outputType, "json", "yaml"); err != nil {
			return err
		}
		mgmntClient := utils.GetSuprSendMgmntClient()
		spinner := utils.NewSpinner("Getting categories...")

		categoriesResp, err := mgmntClient.ListCategories(workspace, mode)
		if err != nil {
			spinner.Stop("")
			log.WithError(err).Errorf("Error getting categories")
			return clierr.Wrap(err, clierr.CodeAPIInternal, "")
		}

		localesResp, err := mgmntClient.ListPreferenceTranslations(workspace)
		if err != nil {
			spinner.Stop("")
			log.WithError(err).Errorf("Error listing preference translations")
			return clierr.Wrap(err, clierr.CodeAPIInternal, "")
		}

		translations := map[string]any{}
		for _, item := range localesResp.Results {
			locale := item.Locale
			if locale == "en" {
				continue
			}
			content, err := mgmntClient.GetPreferenceTranslationsForLocale(workspace, locale)
			if err != nil {
				spinner.Stop("")
				log.WithError(err).Errorf("Error getting translations for locale %s", locale)
				return clierr.Wrap(err, clierr.CodeAPIInternal, "")
			}
			translations[locale] = content
		}

		spinner.Stop(fmt.Sprintf("Successfully got categories for '%s'", workspace))

		output := map[string]any{
			"categories":   categoriesResp,
			"translations": translations,
		}
		utils.OutputData(output, outputType)
		return nil
	},
}

func init() {
	categoryGetCmd.PersistentFlags().StringP("mode", "m", "live", "Version mode: draft or live")
	categoryGetCmd.PersistentFlags().StringP("output", "o", "json", "Output format: json or yaml")
	CategoryCmd.AddCommand(categoryGetCmd)
}
