package translation

import (
	"context"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/utils"
	"github.com/yarlson/pin"
)

var translationGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get translations",
	Long:  "Get all translations for the workspace. Example: suprsend translation get",
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		mode, _ := cmd.Flags().GetString("mode")
		outputType, _ := cmd.Flags().GetString("output")
		mgmntClient := utils.GetSuprSendMgmntClient()
		var p *pin.Pin
		var cancel context.CancelFunc
		if !utils.IsOutputPiped() {
			p = pin.New("Getting translations...",
				pin.WithSpinnerColor(pin.ColorCyan),
				pin.WithTextColor(pin.ColorYellow),
			)
			cancel = p.Start(context.Background())
		}

		translationsResp, err := mgmntClient.GetTranslations(workspace, mode)
		if err != nil {
			if p != nil {
				p.Stop("")
				cancel()
			}
			log.WithError(err).Errorf("Error getting translations")
			return err
		}

		output := map[string]any{}
		for _, item := range translationsResp.Results {
			obj, ok := item.(map[string]any)
			if !ok {
				continue
			}
			filename, _ := obj["filename"].(string)
			locale := strings.TrimSuffix(filename, ".json")
			output[locale] = obj["content"]
		}

		if p != nil {
			p.Stop(fmt.Sprintf("Successfully got %d translation(s)", len(output)))
			cancel()
		}

		utils.OutputData(output, outputType)
		return nil
	},
}

func init() {
	translationGetCmd.PersistentFlags().String("mode", "live", "mode to fetch translations from.")
	translationGetCmd.PersistentFlags().StringP("output", "o", "json", "Output format (json, yaml)")
	TranslationCmd.AddCommand(translationGetCmd)
}
