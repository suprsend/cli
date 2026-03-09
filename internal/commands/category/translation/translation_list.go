package translation

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/utils"
	"github.com/yarlson/pin"
)

var translationListCmd = &cobra.Command{
	Use:   "list",
	Short: "List preference translations",
	Long:  "List preference translations",
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		outputType, _ := cmd.Flags().GetString("output")

		return listTranslations(workspace, outputType)
	},
}

func listTranslations(workspace, outputType string) error {
	if workspace == "" {
		return fmt.Errorf("workspace flag is required")
	}

	mgmntClient := utils.GetSuprSendMgmntClient()

	var p *pin.Pin
	if !utils.IsOutputPiped() {
		p = pin.New("Loading...",
			pin.WithSpinnerColor(pin.ColorCyan),
			pin.WithTextColor(pin.ColorYellow),
		)
		cancel := p.Start(context.Background())
		defer cancel()
	}

	translations, err := mgmntClient.ListPreferenceTranslations(workspace)
	if err != nil {
		return fmt.Errorf("couldn't fetch translations: %w", err)
	}

	msg := fmt.Sprintf("Listed %d translation locales from %s", len(translations.Results), workspace)
	if p != nil {
		p.Stop(msg)
	}

	if len(translations.Results) == 0 && utils.IsOutputPiped() {
		utils.OutputData([]interface{}{}, outputType)
		return nil
	}
	utils.OutputData(translations.Results, outputType)
	return nil
}

func init() {
	TranslationCmd.AddCommand(translationListCmd)
}
