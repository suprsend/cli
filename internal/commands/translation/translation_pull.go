package translation

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

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
					fmt.Fprintf(os.Stdout, "Using default directory: %s\n", outputDir)
				} else {
					outputDir = promptForOutputDirectory()
				}
			}
			if outputDir == "" {
				fmt.Fprintf(os.Stdout, "No output directory specified. Exiting \n")
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
			fmt.Fprintf(os.Stdout, "Error: Failed to get translations: %v\n", err)
			return
		}
		if p != nil {
			p.Stop(fmt.Sprintf("Pulled %d translations from %s", len(translationResp.Results), workspace))
		}

		stats, err := WriteTranslationToFiles(*translationResp, outputDir)
		if err != nil {
			fmt.Fprintf(os.Stdout, "Error: Failed to save translations: %v\n", err)
			return
		}

		fmt.Fprintf(os.Stdout, "\n=== Translation Pull Summary ===\n")
		fmt.Fprintf(os.Stdout, "Total translations processed: %d\n", stats.Total)
		fmt.Fprintf(os.Stdout, "Successfully written: %d\n", stats.Success)
		fmt.Fprintf(os.Stdout, "Failed to write: %d", stats.Failed)

		if stats.Failed > 0 {
			fmt.Fprintf(os.Stdout, "\nFailed translations:\n")
			for _, errorMsg := range stats.Errors {
				fmt.Fprintf(os.Stdout, " - %s\n", errorMsg)
			}
		}
	},
}

func init() {
	translationPullCmd.PersistentFlags().StringP("mode", "m", "live", "Version mode: draft or live")
	translationPullCmd.PersistentFlags().BoolP("force", "f", false, "Skip directory confirmation prompt, use default path")
	translationPullCmd.PersistentFlags().StringP("dir", "d", "", "Directory to save translation files to")
	TranslationCmd.AddCommand(translationPullCmd)
}
