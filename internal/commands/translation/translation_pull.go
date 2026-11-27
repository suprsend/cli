package translation

import (
	"errors"
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

		report := utils.NewReport("translation pull", workspace, false)
		spinner := utils.NewSpinner("Loading...")

		mgmnt_client := utils.GetSuprSendMgmntClient()
		translationResp, err := mgmnt_client.GetTranslations(workspace, mode)
		spinner.Stop("")
		if err != nil {
			return report.Finish(clierr.Wrap(err, clierr.CodeAPIInternal, ""))
		}

		stats, err := WriteTranslationToFiles(*translationResp, outputDir)
		if err != nil {
			return report.Finish(clierr.Wrap(err, clierr.CodeFileParseFailed, ""))
		}
		for _, name := range stats.Written {
			report.Add("translations/"+name, utils.StatusPulled, "")
		}
		for _, e := range stats.Errors {
			report.Fail("translations", errors.New(e))
		}
		if stats.Failed > 0 {
			return report.Finish(clierr.Failures(fmt.Sprintf("%d translation(s) failed to write", stats.Failed), stats.Errors, clierr.CodeAPIInternal))
		}
		return report.Finish(nil)
	},
}

func init() {
	translationPullCmd.PersistentFlags().StringP("mode", "m", "live", "Version mode: draft or live")
	translationPullCmd.PersistentFlags().BoolP("force", "F", false, "Skip directory confirmation prompt, use default path")
	translationPullCmd.PersistentFlags().StringP("dir", "d", "", "Directory to save translation files to (default: ./suprsend/translations)")
	TranslationCmd.AddCommand(translationPullCmd)
}
