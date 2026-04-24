package translation

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/utils"
)

var translationListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Translations",
	Long:  "List template translation files in a workspace. Returns translation file names and metadata. Use --mode to switch between draft and live versions.",
	Annotations: map[string]string{
		"skills:tip:output": "Use `-o json` for machine-readable JSON output, `-o yaml` for YAML. Default `-o pretty` outputs a human-friendly table.",
	},
	Run: func(cmd *cobra.Command, args []string) {
		spinner := utils.NewSpinner("Loading...")
		mode, _ := cmd.Flags().GetString("mode")
		workspace, _ := cmd.Flags().GetString("workspace")
		includeContent, _ := cmd.Flags().GetString("include-content")
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")
		mgmntClient := utils.GetSuprSendMgmntClient()
		translations, err := mgmntClient.ListTranslations(workspace, mode, includeContent, limit, offset)
		if err != nil {
			log.WithError(err).Error("Couldn't fetch translations")
			return
		}
		spinner.Stop(fmt.Sprintf("Listed %d translation files from %s in %s mode", len(translations.Results), workspace, mode))
		outputType, _ := cmd.Flags().GetString("output")
		if len(translations.Results) == 0 && utils.IsOutputPiped() {
			utils.OutputData([]interface{}{}, outputType)
			return
		}
		utils.OutputData(translations.Results, outputType)
	},
}

func init() {
	translationListCmd.Flags().String("include-content", "false", "Include translation file content in the response (true/false)")
	translationListCmd.Flags().IntP("limit", "l", 20, "Maximum number of translations to return")
	translationListCmd.Flags().Int("offset", 0, "Number of translations to skip for pagination")
	translationListCmd.Flags().StringP("mode", "m", "live", "Version mode: draft or live")
	translationListCmd.Flags().StringP("output", "o", "pretty", "Output format: pretty, json, or yaml")
	TranslationCmd.PersistentFlags().StringP("workspace", "w", "staging", "Workspace name (e.g., staging, production)")
	TranslationCmd.PersistentFlags().StringP("service-token", "s", "", "Service token (default: $SUPRSEND_SERVICE_TOKEN)")
	TranslationCmd.AddCommand(translationListCmd)
}
