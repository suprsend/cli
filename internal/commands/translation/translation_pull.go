package translation

import (
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/utils"
)

var translationPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull translation files",
	Long:  "Download template translation files from a workspace to local JSON files. Saves one JSON file per translation, named by its filename (e.g. `candidate.es.json`), to the output directory (default `./suprsend/translations`), overwriting existing files with the same name.",
	Example: `  # Pull all translations to default directory (suprsend/translations/)
  suprsend translation pull

  # Pull to a custom directory
  suprsend translation pull --dir ./my-translations

  # Pull draft translations
  suprsend translation pull --mode draft`,
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		mode, _ := cmd.Flags().GetString("mode")
		outputDir, _ := cmd.Flags().GetString("dir")
		force, _ := cmd.Flags().GetBool("force")
		if outputDir == "" {
			outputDir = filepath.Join(".", "suprsend", "translations")
			if _, err := os.Stat(outputDir); os.IsNotExist(err) {
				if force {
					log.Infof("Using default directory: %s", outputDir)
				} else {
					od, success := promptForOutputDirectory()
					if !success {
						return clierr.New("output directory is required; pass --dir, or --force to use the default", clierr.CodeInvalidUsage)
					}
					outputDir = od
				}
			}
			if outputDir == "" {
				log.Info("No output directory specified. Exiting.")
				return nil
			}
		}

		spinner := utils.NewSpinner("Loading...")

		mgmnt_client := utils.GetSuprSendMgmntClient()
		translationResp, err := mgmnt_client.GetTranslations(workspace, mode)
		if err != nil {
			spinner.Stop("")
			log.Errorf("Failed to get translations: %v", err)
			return clierr.Wrap(err, clierr.CodeAPIInternal, "")
		}
		spinner.Stop(fmt.Sprintf("Pulled %d translations from %s", len(translationResp.Results), workspace))

		stats, err := WriteTranslationToFiles(*translationResp, outputDir)
		if err != nil {
			log.Errorf("Failed to save translations: %v", err)
			return clierr.Wrap(err, clierr.CodeFileParseFailed, "")
		}

		log.Info("=== Translation Pull Summary ===")
		log.Infof("Total translations processed: %d", stats.Total)
		log.Infof("Successfully written: %d", stats.Success)
		log.Infof("Failed to write: %d", stats.Failed)

		if stats.Failed > 0 {
			log.Info("Failed translations:")
			for _, errorMsg := range stats.Errors {
				log.Infof("  - %s", errorMsg)
			}
			return clierr.Failures(fmt.Sprintf("%d translation(s) failed to write", stats.Failed), stats.Errors, clierr.CodeAPIInternal)
		}
		return nil
	},
}

func init() {
	translationPullCmd.PersistentFlags().StringP("mode", "m", "live", "Version mode: draft or live")
	translationPullCmd.PersistentFlags().BoolP("force", "F", false, "Skip directory confirmation prompt, use default path")
	translationPullCmd.PersistentFlags().StringP("dir", "d", "", "Directory to save translation files to (default: ./suprsend/translations)")
	TranslationCmd.AddCommand(translationPullCmd)
}
