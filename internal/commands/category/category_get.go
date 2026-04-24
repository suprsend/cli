package category

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/utils"
	"github.com/yarlson/pin"
)

var categoryGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get categories and translations",
	Long:  "Retrieve preference categories and their translations from a workspace. Returns the full category structure along with translations for all non-English locales. Use --mode to switch between draft and live versions.",
	Annotations: map[string]string{
		"skills:tip:output": "Use `-o json` for machine-readable JSON output, `-o yaml` for YAML.",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		mode, _ := cmd.Flags().GetString("mode")
		outputType, _ := cmd.Flags().GetString("output")
		mgmntClient := utils.GetSuprSendMgmntClient()
		var p *pin.Pin
		var cancel context.CancelFunc
		if !utils.IsOutputPiped() {
			p = pin.New("Getting categories...",
				pin.WithSpinnerColor(pin.ColorCyan),
				pin.WithTextColor(pin.ColorYellow),
			)
			cancel = p.Start(context.Background())
		}

		categoriesResp, err := mgmntClient.ListCategories(workspace, mode)
		if err != nil {
			if p != nil {
				p.Stop("")
				cancel()
			}
			log.WithError(err).Errorf("Error getting categories")
			return err
		}

		localesResp, err := mgmntClient.ListPreferenceTranslations(workspace)
		if err != nil {
			if p != nil {
				p.Stop("")
				cancel()
			}
			log.WithError(err).Errorf("Error listing preference translations")
			return err
		}

		translations := map[string]any{}
		for _, item := range localesResp.Results {
			locale := item.Locale
			if locale == "en" {
				continue
			}
			content, err := mgmntClient.GetPreferenceTranslationsForLocale(workspace, locale)
			if err != nil {
				if p != nil {
					p.Stop("")
					cancel()
				}
				log.WithError(err).Errorf("Error getting translations for locale %s", locale)
				return err
			}
			translations[locale] = content
		}

		if p != nil {
			p.Stop(fmt.Sprintf("Successfully got categories for '%s'", workspace))
			cancel()
		}

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
