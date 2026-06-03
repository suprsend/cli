package category

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/commands/category/translation"
	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/cli/mgmnt"
)

type jsonCategoryInput struct {
	Categories   *CategoriesOnDisk                             `json:"categories"`
	Translations map[string]mgmnt.PreferenceTranslationContent `json:"translations"`
}

var categoryPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push categories to a workspace",
	Long:  `Upload local preference categories and translations to a workspace. Reads categories_preferences.json and translation files from the input directory. By default, changes are staged as drafts. Use --commit to also promote to live.`,
	Example: `  # Push from local files (default directory)
  suprsend category push

  # Push from a custom directory
  suprsend category push --dir ./my-categories

  # Push and commit to live immediately
  suprsend category push --commit

  # Push categories inline via JSON
  suprsend category push --json '{"categories": {...}}'`,
	Annotations: map[string]string{
		"skills:tip.a-draft":  "Push writes to the **draft** state. Run `suprsend category commit` to promote draft → live.",
		"skills:tip.b-dryrun": "Pair with `--dry-run` to validate the categories server-side without writing to the draft. Pair with `--commit` to push + commit in one step.",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		path, _ := cmd.Flags().GetString("dir")
		commit, _ := cmd.Flags().GetBool("commit")
		commitMessage, _ := cmd.Flags().GetString("commit-message")
		jsonPayload, _ := cmd.Flags().GetString("json")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		force, _ := cmd.Flags().GetBool("force")

		if commit && !dryRun && !force {
			msg := fmt.Sprintf("This will push and promote categories to live in workspace \"%s\". Continue?", workspace)
			confirmed, err := utils.ConfirmDestructiveAction(msg)
			if err != nil || !confirmed {
				log.Info("Aborted.")
				return nil
			}
		}

		// Aggregate stats across translations + categories so the run finishes
		// with a single "=== Category Push Summary ===" block matching the
		// shape used by schema push. Translations contribute one item per
		// locale; the categories tree contributes exactly one item.
		translationStats := &translation.PushTranslationStats{}
		categorySuccess := false
		categoryFailErr := ""

		if jsonPayload != "" {
			var input jsonCategoryInput
			if err := json.Unmarshal([]byte(jsonPayload), &input); err != nil {
				return clierr.Wrap(err, clierr.CodeFileParseFailed, "")
			}
			if input.Categories == nil {
				return clierr.New("--json payload missing required \"categories\" field", clierr.CodeUnknown)
			}

			mgmntClient := utils.GetSuprSendMgmntClient()

			pushableLocales := 0
			for locale := range input.Translations {
				if locale != "en" {
					pushableLocales++
				}
			}

			if dryRun {
				sectionCount, categoryCount := 0, 0
				for _, rc := range input.Categories.RootCategories {
					sectionCount += len(rc.Sections)
					for _, s := range rc.Sections {
						categoryCount += len(s.Categories)
					}
				}
				log.Infof("DRY RUN: would push %d section%s, %d categor%s and %d translation(s) to %s",
					sectionCount, pluralS(sectionCount),
					categoryCount, pluralIes(categoryCount),
					pushableLocales, workspace,
				)
				return nil
			}

			// Translations don't have a draft/live distinction — push them
			// regardless of --commit so they always reflect the local state.
			translationStats.Total = pushableLocales
			for locale, t := range input.Translations {
				if locale == "en" {
					translationStats.SkippedEnglish++
					continue
				}
				spinner := utils.NewSpinner(fmt.Sprintf("Pushing %s.json...", locale))
				if err := mgmntClient.PushPreferenceTranslation(cmd.Context(), workspace, locale, t); err != nil {
					spinner.Stop("")
					log.WithError(err).Errorf("preference_categories/translations/%s.json: failed to push", locale)
					translationStats.Failed++
					translationStats.Errors = append(translationStats.Errors, fmt.Sprintf("preference_categories/translations/%s.json: failed to push: %v", locale, err))
					continue
				}
				translationStats.Success++
				spinner.Stop(fmt.Sprintf("Pushed translation: %s.json", locale))
			}

			catSpinner := utils.NewSpinner("Pushing categories...")
			if err := mgmntClient.PushCategories(cmd.Context(), workspace, input.Categories, commit, commitMessage); err != nil {
				catSpinner.Stop("")
				log.WithError(err).Error("preference_categories/categories.json: failed to push")
				categoryFailErr = err.Error()
			} else {
				categorySuccess = true
				catSpinner.Stop("Pushed categories")
			}

			sectionCount, categoryCount := countSectionsAndCategories(input.Categories)
			emitCategoryPushSummary(translationStats, categorySuccess, categoryFailErr, sectionCount, categoryCount)
			if !categorySuccess || translationStats.Failed > 0 {
				return clierr.New("category push had errors", clierr.CodeAPIInternal)
			}
			return nil
		}

		translationDir := path
		if translationDir == "" {
			translationDir = filepath.Join(".", defaultCategoryDir)
		}
		translationDir = filepath.Join(translationDir, "translations")

		if path == "" {
			path = filepath.Join(".", defaultCategoryDir, "categories.json")
		} else {
			path = filepath.Join(path, "categories.json")
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			log.Errorf("Directory %s does not exist", path)
			return clierr.Wrap(err, clierr.CodeFileNotFound, "")
		}

		categories, err := ReadFromFile(path)
		if err != nil {
			log.WithError(err).Error("Couldn't read categories from file")
			return clierr.Wrap(err, clierr.CodeFileParseFailed, "")
		}

		// Push translations first, regardless of --commit. They don't have a
		// draft/live model, so any local change should land immediately.
		// English-only / no-files cases get demoted from error to debug log so
		// they don't fail the broader category push.
		ts, terr := translation.PushTranslations(cmd.Context(), workspace, "", translationDir, dryRun)
		if ts != nil {
			translationStats = ts
		}
		if terr != nil {
			var ce *clierr.CLIError
			if errors.As(terr, &ce) && ce.Code == clierr.CodeInvalidUsage {
				log.Debugf("No translations to push: %v", terr)
			} else if errors.As(terr, &ce) && ce.Code == clierr.CodeFileNotFound {
				log.Debugf("No translation files found: %v", terr)
			} else {
				log.WithError(terr).Warn("Translation push had errors; continuing with categories")
			}
		}

		if dryRun {
			sectionCount, categoryCount := countSectionsAndCategories(categories)
			log.Infof("DRY RUN: would push %d section%s, %d categor%s and %d translation(s) to %s",
				sectionCount, pluralS(sectionCount),
				categoryCount, pluralIes(categoryCount),
				translationStats.Total, workspace,
			)
			return nil
		}

		spinner2 := utils.NewSpinner("Pushing categories...")
		mgmnt_client := utils.GetSuprSendMgmntClient()
		err = mgmnt_client.PushCategories(cmd.Context(), workspace, categories, commit, commitMessage)
		if err != nil {
			spinner2.Stop("")
			log.WithError(err).Error("preference_categories/categories.json: failed to push")
			categoryFailErr = err.Error()
		} else {
			categorySuccess = true
			spinner2.Stop("Pushed categories")
		}

		sectionCount, categoryCount := 0, 0
		for _, rc := range categories.RootCategories {
			sectionCount += len(rc.Sections)
			for _, s := range rc.Sections {
				categoryCount += len(s.Categories)
			}
		}
		emitCategoryPushSummary(translationStats, categorySuccess, categoryFailErr, sectionCount, categoryCount)
		if !categorySuccess || translationStats.Failed > 0 {
			return clierr.New("category push had errors", clierr.CodeAPIInternal)
		}
		return nil
	},
}

// emitCategoryPushSummary prints the end-of-run summary for `category push`.
// The "Category Push Summary" block is always printed and reports the
// section/category counts plus push outcome. The "Translation Push Summary"
// block follows only when translations were attempted or English files were
// skipped (i.e. t.Total or t.SkippedEnglish is non-zero).
func emitCategoryPushSummary(t *translation.PushTranslationStats, categorySuccess bool, categoryFailErr string, sectionCount, categoryCount int) {
	if t == nil {
		t = &translation.PushTranslationStats{}
	}

	log.Info("=== Category Push Summary ===")
	log.Infof("Sections: %d", sectionCount)
	log.Infof("Categories: %d", categoryCount)
	if categorySuccess {
		log.Info("Successfully pushed")
	} else {
		log.Infof("Failed to push: %s", categoryFailErr)
	}

	if t.Total > 0 || t.SkippedEnglish > 0 {
		log.Info("=== Translation Push Summary ===")
		log.Infof("Total locales processed: %d", t.Total)
		log.Infof("Successfully pushed: %d", t.Success)
		log.Infof("Failed to push: %d", t.Failed)
		if t.SkippedEnglish > 0 {
			log.Infof("Skipped (English source of truth): %d", t.SkippedEnglish)
		}
		if len(t.Errors) > 0 {
			log.Info("Errors:")
			for _, e := range t.Errors {
				log.Infof("  - %s", e)
			}
		}
	}
}

func countSectionsAndCategories(cats *CategoriesOnDisk) (int, int) {
	if cats == nil {
		return 0, 0
	}
	sectionCount, categoryCount := 0, 0
	for _, rc := range cats.RootCategories {
		sectionCount += len(rc.Sections)
		for _, s := range rc.Sections {
			categoryCount += len(s.Categories)
		}
	}
	return sectionCount, categoryCount
}

func pluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func pluralIes(n int) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}

func init() {
	categoryPushCmd.Flags().StringP("dir", "d", "", "Directory containing category files (default: ./"+defaultCategoryDir+")")
	categoryPushCmd.PersistentFlags().BoolP("commit", "c", false, "Promote changes from draft to live after pushing")
	categoryPushCmd.PersistentFlags().String("commit-message", "", "Message describing the changes being committed")
	categoryPushCmd.Flags().StringP("json", "j", "", `Categories (and optional translations) as a JSON object. Required "categories" key holds the preference category structure. Optional "translations" key maps locale codes to objects with "sections" and "categories" keys, e.g. '{"categories":{"root_categories":[...]},"translations":{"es":{"sections":{"key":{"name":"...","description":"..."}},"categories":{"key":{"name":"...","description":"..."}}}}}'`)
	categoryPushCmd.Flags().BoolP("dry-run", "n", false, "Print what would be pushed without making any changes")
	categoryPushCmd.Flags().BoolP("force", "F", false, "Skip confirmation prompt when --commit is set")
	CategoryCmd.AddCommand(categoryPushCmd)
}
