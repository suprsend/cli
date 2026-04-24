package translation

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/utils"
	"github.com/yarlson/pin"
)

var translationPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull Translation files",
	Long:  "Download template translation files from a workspace to local JSON files. Saves one JSON file per translation to the output directory.",
	Run: func(cmd *cobra.Command, args []string) {
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
						return
					}
					outputDir = od
				}
			}
			if outputDir == "" {
				log.Info("No output directory specified. Exiting.")
				return
			}
		}

		var p *pin.Pin
		if !utils.IsOutputPiped() {
			p = pin.New("Loading...",
				pin.WithSpinnerColor(pin.ColorCyan),
				pin.WithTextColor(pin.ColorYellow),
			)
			cancel := p.Start(context.Background())
			defer cancel()
		}

		mgmnt_client := utils.GetSuprSendMgmntClient()
		translationResp, err := mgmnt_client.GetTranslations(workspace, mode)
		if err != nil {
			log.Errorf("Failed to get translations: %v", err)
			return
		}
		if p != nil {
			p.Stop(fmt.Sprintf("Pulled %d translations from %s", len(translationResp.Results), workspace))
		}

		stats, err := WriteTranslationToFiles(*translationResp, outputDir)
		if err != nil {
			log.Errorf("Failed to save translations: %v", err)
			return
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
		}
	},
}

func init() {
	translationPullCmd.PersistentFlags().StringP("mode", "m", "live", "Version mode: draft or live")
	translationPullCmd.PersistentFlags().BoolP("force", "F", false, "Skip directory confirmation prompt, use default path")
	translationPullCmd.PersistentFlags().StringP("dir", "d", "", "Directory to save translation files to")
	TranslationCmd.AddCommand(translationPullCmd)
}
