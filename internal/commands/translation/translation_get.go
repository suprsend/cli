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
	Long:  "Retrieve all template translations from a workspace. Returns translation content keyed by locale code. Use --mode to switch between draft and live versions.",
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
	translationGetCmd.PersistentFlags().StringP("mode", "m", "live", "Version mode: draft or live")
	translationGetCmd.PersistentFlags().StringP("output", "o", "json", "Output format: json or yaml")
	TranslationCmd.AddCommand(translationGetCmd)
}
