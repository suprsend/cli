package translation

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/utils"
	"github.com/yarlson/pin"
)

var translationListCmd = &cobra.Command{
	Use:   "list",
	Short: "List preference translations",
	Long:  "List preference translations",
	Run: func(cmd *cobra.Command, args []string) {
		workspace, _ := cmd.Flags().GetString("workspace")
		outputType, _ := cmd.Flags().GetString("output")

		listTranslations(workspace, outputType)
	},
}

func listTranslations(workspace, outputType string) {
	if workspace == "" {
		log.Error("workspace flag is required")
		return
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
		log.WithError(err).Error("Couldn't fetch translations")
		return
	}

	msg := fmt.Sprintf("Listed %d translation locales from %s", len(translations.Results), workspace)
	if p != nil {
		p.Stop(msg)
	}

	if len(translations.Results) == 0 && utils.IsOutputPiped() {
		utils.OutputData([]interface{}{}, outputType)
		return
	}
	utils.OutputData(translations.Results, outputType)
}

func init() {
	TranslationCmd.AddCommand(translationListCmd)
}
