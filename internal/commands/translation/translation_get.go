package translation

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/utils"
)

var translationGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get translations",
	Long:  "Retrieve all template translations from a workspace. Returns translation content keyed by locale code. Use --mode to switch between draft and live versions.",
	Example: `  # Get all translations (live mode)
  suprsend translation get

  # Get draft translations
  suprsend translation get --mode draft

  # Get with JSON output
  suprsend translation get --output json`,
	Annotations: map[string]string{
		"skills:tip:output": "Use `-o json` for machine-readable JSON output, `-o yaml` for YAML.",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		mode, _ := cmd.Flags().GetString("mode")
		outputType, _ := cmd.Flags().GetString("output")
		mgmntClient := utils.GetSuprSendMgmntClient()
		spinner := utils.NewSpinner("Getting translations...")

		translationsResp, err := mgmntClient.GetTranslations(workspace, mode)
		if err != nil {
			spinner.Stop("")
			log.WithError(err).Errorf("Error getting translations")
			return clierr.Wrap(err, clierr.CodeAPIInternal, "")
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

		spinner.Stop(fmt.Sprintf("Successfully got %d translation(s)", len(output)))

		utils.OutputData(output, outputType)
		return nil
	},
}

func init() {
	translationGetCmd.PersistentFlags().StringP("mode", "m", "live", "Version mode: draft or live")
	TranslationCmd.AddCommand(translationGetCmd)
}
