package category

import (
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/commands/category/translation"
	"github.com/suprsend/cli/internal/utils"
)

var categoryPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull categories from a workspace",
	Long:  "Download preference categories and their translations from a workspace to local files. Saves categories_preferences.json and locale-specific translation files to the output directory.",
	Example: `  # Pull categories to default directory (suprsend/categories/)
  suprsend category pull

  # Pull to a custom directory
  suprsend category pull --dir ./my-categories

  # Pull draft categories
  suprsend category pull --mode draft`,
	Annotations: map[string]string{
		"skills:tip.a-overwrite": "Pull overwrites local `categories_preferences.json` and translation files. Commit local edits first if you don't want them clobbered (or use `--force` to skip the prompt).",
		"skills:tip.b-mode":      "Defaults to the **live** mode. Use `--mode draft` to mirror the pending state instead.",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		mode, _ := cmd.Flags().GetString("mode")
		outputDir, _ := cmd.Flags().GetString("dir")
		force, _ := cmd.Flags().GetBool("force")
		if outputDir == "" {
			outputDir = filepath.Join(".", defaultCategoryDir)
			if _, err := os.Stat(outputDir); os.IsNotExist(err) {
				if force {
					log.Infof("Using default directory: %s", outputDir)
				} else {
					od, success := promptForOutputDirectory()
					if !success {
						log.Info("No output directory specified. Exiting.")
						return nil
					}
					outputDir = od
				}
			}
			if outputDir == "" {
				log.Info("No output directory specified. Exiting.")
				return nil
			}
		}
		if err := ensureOutputDirectory(outputDir); err != nil {
			log.Errorf("Error with output directory: %v", err)
			return clierr.Wrap(err, clierr.CodeFileNotFound, "")
		}
		spinner := utils.NewSpinner("Loading...")

		mgmntClient := utils.GetSuprSendMgmntClient()
		categories, err := mgmntClient.ListCategories(cmd.Context(), workspace, mode)
		if err != nil {
			log.WithError(err).Error("Couldn't fetch categories")
			return clierr.Wrap(err, clierr.CodeAPIInternal, "")
		}
		filePath := filepath.Join(outputDir, "categories.json")
		spinner.Stop(fmt.Sprintf("Pulled categories from %s", workspace))
		err = writeCategoriesFile(categories, filePath)
		if err != nil {
			log.WithError(err).Error("Couldn't write categories to file")
			return clierr.Wrap(err, clierr.CodeFileParseFailed, "")
		}

		totalSections := 0
		totalCategories := 0
		for _, rc := range categories.RootCategories {
			totalSections += len(rc.Sections)
			for _, s := range rc.Sections {
				totalCategories += len(s.Categories)
			}
		}
		log.Info("=== Category Pull Summary ===")
		log.Infof("Sections: %d", totalSections)
		log.Infof("Categories: %d", totalCategories)
		log.Infof("Written to: %s", filePath)

		translationDir := filepath.Join(outputDir, "translations")
		if err := translation.PullTranslations(cmd.Context(), workspace, translationDir, force); err != nil {
			return clierr.Wrap(err, clierr.CodeAPIInternal, "")
		}

		return nil
	},
}

func init() {
	categoryPullCmd.PersistentFlags().StringP("mode", "m", "live", "Version mode: draft or live")
	categoryPullCmd.Flags().StringP("dir", "d", "", "Directory to save category files to (default: ./"+defaultCategoryDir+")")
	categoryPullCmd.PersistentFlags().BoolP("force", "F", false, "Skip directory confirmation prompt, use default path")
	CategoryCmd.AddCommand(categoryPullCmd)
}
