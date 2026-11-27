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
	Long:  "Download preference categories and their translations from a workspace to local files. Writes `categories.json` and one `translations/<locale>.json` file per locale to the output directory (default `./suprsend/preference_categories`).",
	Example: `  # Pull categories to default directory (suprsend/preference_categories/)
  suprsend category pull

  # Pull to a custom directory
  suprsend category pull --dir ./my-categories

  # Pull draft categories
  suprsend category pull --mode draft`,
	Annotations: map[string]string{
		"skills:tip.a-overwrite": "Pull overwrites the local `categories.json` and `translations/<locale>.json` files without asking, so commit or stash local edits first. `--force` does not affect overwriting: it only skips the output-directory prompt shown when `--dir` is omitted and `./suprsend/preference_categories` doesn't exist. In non-interactive shells pass `--dir` or `--force`; otherwise the command exits 64 (invalid usage) without pulling anything.",
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
		if err := ensureOutputDirectory(outputDir); err != nil {
			return clierr.Wrap(err, clierr.CodeFileNotFound, "")
		}
		report := utils.NewReport("category pull", workspace, false)
		spinner := utils.NewSpinner("Loading...")

		mgmntClient := utils.GetSuprSendMgmntClient()
		categories, err := mgmntClient.ListCategories(workspace, mode)
		spinner.Stop("")
		if err != nil {
			return report.Finish(clierr.Wrap(err, clierr.CodeAPIInternal, ""))
		}
		filePath := filepath.Join(outputDir, "categories.json")
		if err := writeCategoriesFile(categories, filePath); err != nil {
			err = clierr.Wrap(err, clierr.CodeFileParseFailed, "")
			report.Fail("preference_categories", err)
			return report.Finish(err)
		}

		totalSections := 0
		totalCategories := 0
		for _, rc := range categories.RootCategories {
			totalSections += len(rc.Sections)
			for _, s := range rc.Sections {
				totalCategories += len(s.Categories)
			}
		}
		report.Add("preference_categories", utils.StatusPulled, fmt.Sprintf("%d section%s, %d categor%s", totalSections, pluralS(totalSections), totalCategories, pluralIes(totalCategories)))

		translationDir := filepath.Join(outputDir, "translations")
		if err := translation.PullTranslations(report, workspace, translationDir, force); err != nil {
			return report.Finish(clierr.Wrap(err, clierr.CodeAPIInternal, ""))
		}
		return report.Finish(nil)
	},
}

func init() {
	categoryPullCmd.PersistentFlags().StringP("mode", "m", "live", "Version mode: draft or live")
	categoryPullCmd.Flags().StringP("dir", "d", "", "Directory to save category files to (default: ./"+defaultCategoryDir+")")
	categoryPullCmd.PersistentFlags().BoolP("force", "F", false, "Skip directory confirmation prompt, use default path")
	CategoryCmd.AddCommand(categoryPullCmd)
}
