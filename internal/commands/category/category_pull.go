package category

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/commands/category/translation"
	"github.com/suprsend/cli/internal/utils"
	"github.com/yarlson/pin"
)

var categoryPullCmd = &cobra.Command{
	Use:   "pull",
	Long:  "Pull categories from a workspace",
	Short: "Pull categories from a workspace",
	Run: func(cmd *cobra.Command, args []string) {
		workspace, _ := cmd.Flags().GetString("workspace")
		mode, _ := cmd.Flags().GetString("mode")
		outputDir, _ := cmd.Flags().GetString("dir")
		force, _ := cmd.Flags().GetBool("force")
		if outputDir == "" {
			outputDir = filepath.Join(".", "suprsend", "category")
			if _, err := os.Stat(outputDir); os.IsNotExist(err) {
				if force {
					fmt.Fprintf(os.Stdout, "Using default directory: %s\n", outputDir)
				} else {
					outputDir = promptForOutputDirectory()
				}
			}
			if outputDir == "" {
				fmt.Fprintf(os.Stdout, "No output directory specified. Exiting.\n")
				return
			}
		}
		if err := ensureOutputDirectory(outputDir); err != nil {
			fmt.Fprintf(os.Stdout, "Error with output directory: %v\n", err)
			return
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

		mgmntClient := utils.GetSuprSendMgmntClient()
		categories, err := mgmntClient.ListCategories(workspace, mode)
		if err != nil {
			log.WithError(err).Error("Couldn't fetch categories")
			return
		}
		filePath := filepath.Join(outputDir, "categories_preferences.json")
		if p != nil {
			p.Stop(fmt.Sprintf("Pulled categories from %s", workspace))
		}
		err = WriteToFileWithPath(categories, filePath)
		if err != nil {
			log.WithError(err).Error("Couldn't write categories to file")
			return
		}

		translationDir := filepath.Join(outputDir, "translation")
		translation.PullTranslations(workspace, translationDir, force)
	},
}

func init() {
	categoryPullCmd.PersistentFlags().StringP("mode", "m", "live", "Mode to pull categories from")
	categoryPullCmd.Flags().StringP("dir", "d", "", "Output directory for categories (default: ./suprsend/category)")
	categoryPullCmd.PersistentFlags().BoolP("force", "f", false, "Force using default directory without prompting")
	CategoryCmd.AddCommand(categoryPullCmd)
}
