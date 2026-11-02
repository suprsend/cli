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
	Long:  "Upload local preference categories and translations to a workspace. Reads `categories.json` and `translations/<locale>.json` from the input directory (default `./suprsend/preference_categories`). Categories are staged as a draft unless --commit is passed. Translation files (every locale except `en`) are pushed first and take effect immediately, with or without --commit.",
	Example: `  # Push from local files (default directory)
  suprsend category push

  # Push from a custom directory
  suprsend category push --dir ./my-categories

  # Push and commit to live immediately
  suprsend category push --commit

  # Push categories inline via JSON
  suprsend category push --json '{"categories": {...}}'`,
	Annotations: map[string]string{
		"skills:tip.a-draft":   "Categories are written to the **draft** state. Run `suprsend category commit` to promote draft → live, or pass `--commit` to push + commit in one step.",
		"skills:tip.b-live":    "Translations are not drafted: every non-English `translations/<locale>.json` (or `translations` entry in `--json`) is pushed **live immediately**, with or without `--commit`, before the categories are pushed.",
		"skills:tip.c-dryrun":  "`--dry-run` makes no API calls: it parses `categories.json` (or `--json`), lists the translation files without reading them, and reports counts. A malformed translation file is only caught by a real push. Nothing is validated server-side.",
		"skills:tip.d-confirm": "The `--commit` confirmation prompt only appears when stdin is a TTY. In non-interactive shells (CI, agents) it proceeds without asking, even without `-F`.",
		"skills:tip.e-result":  "The result lists each translation locale and then `preference_categories` as `pushed` (draft only), `committed`, `unchanged` (the draft already matched live) or `failed`, followed by a summary; with `-o json` it is one JSON object with `items` and `counts`.",
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

		report := utils.NewReport("category push", workspace, dryRun)
		mgmntClient := utils.GetSuprSendMgmntClient()
		translationStats := &translation.PushTranslationStats{}

		var categories *CategoriesOnDisk
		if jsonPayload != "" {
			var input jsonCategoryInput
			if err := json.Unmarshal([]byte(jsonPayload), &input); err != nil {
				return clierr.Wrap(err, clierr.CodeFileParseFailed, "")
			}
			if input.Categories == nil {
				return clierr.New("--json payload missing required \"categories\" field", clierr.CodeInvalidUsage)
			}
			categories = input.Categories

			// Translations don't have a draft/live distinction — push them
			// regardless of --commit so they always reflect the local state.
			for locale, t := range input.Translations {
				resource := "preference_categories/translations/" + locale + ".json"
				switch {
				case locale == "en":
					translationStats.SkippedEnglish++
					report.Add(resource, utils.StatusSkipped, "English is the source of truth and can't be pushed")
					continue
				case dryRun:
					translationStats.Total++
					report.Add(resource, utils.StatusWouldPush, "")
					continue
				}
				translationStats.Total++
				spinner := utils.NewSpinner(fmt.Sprintf("Pushing %s.json...", locale))
				err := mgmntClient.PushPreferenceTranslation(workspace, locale, t)
				spinner.Stop("")
				if err != nil {
					translationStats.Failed++
					translationStats.Errors = append(translationStats.Errors, fmt.Sprintf("%s: failed to push: %v", resource, err))
					report.Fail(resource, err)
					continue
				}
				translationStats.Success++
				report.Add(resource, utils.StatusPushed, "live")
			}
		} else {
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
				return clierr.Wrap(err, clierr.CodeFileNotFound, "")
			}

			var err error
			categories, err = ReadFromFile(path)
			if err != nil {
				return clierr.Wrap(err, clierr.CodeFileParseFailed, "")
			}

			// Push translations first, regardless of --commit. They don't have a
			// draft/live model, so any local change should land immediately.
			// English-only / no-files cases get demoted from error to debug log so
			// they don't fail the broader category push.
			ts, terr := translation.PushTranslations(report, workspace, "", translationDir, dryRun)
			if ts != nil {
				translationStats = ts
			}
			if terr != nil {
				var ce *clierr.CLIError
				if errors.As(terr, &ce) && ce.Code == clierr.CodeInvalidUsage {
					log.Debugf("No translations to push: %v", terr)
				} else if errors.As(terr, &ce) && ce.Code == clierr.CodeFileNotFound {
					log.Debugf("No translation files found: %v", terr)
				}
			}
		}

		sectionCount, categoryCount := countSectionsAndCategories(categories)
		counts := fmt.Sprintf("%d section%s, %d categor%s", sectionCount, pluralS(sectionCount), categoryCount, pluralIes(categoryCount))
		if dryRun {
			status := utils.StatusWouldPush
			if commit {
				status = utils.StatusWouldCommit
			}
			report.Add("preference_categories", status, counts)
			return report.Finish(nil)
		}

		spinner := utils.NewSpinner("Pushing categories...")
		commitStatus, err := mgmntClient.PushCategories(workspace, categories, commit, commitMessage)
		spinner.Stop("")
		categoryFailErr := ""
		if err != nil {
			categoryFailErr = err.Error()
			report.Fail("preference_categories", err)
		} else {
			status, detail := utils.PushStatus(commitStatus)
			if detail != "" {
				counts += "; " + detail
			}
			report.Add("preference_categories", status, counts)
		}

		if err != nil || translationStats.Failed > 0 {
			return report.Finish(categoryPushError(translationStats, err == nil, categoryFailErr))
		}
		return report.Finish(nil)
	},
}

// categoryPushError lists everything that failed in a category push: the
// categories themselves and each translation locale.
func categoryPushError(t *translation.PushTranslationStats, categorySuccess bool, categoryFailErr string) error {
	var failures []string
	if !categorySuccess {
		failures = append(failures, clierr.ItemFailure("preference_categories", "failed to push", errors.New(categoryFailErr)))
	}
	failures = append(failures, t.Errors...)
	return clierr.Failures("category push had errors", failures, clierr.CodeAPIInternal)
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
	categoryPushCmd.Flags().BoolP("dry-run", "n", false, "Report what would be pushed without making any changes")
	categoryPushCmd.Flags().BoolP("force", "F", false, "Skip confirmation prompt when --commit is set")
	CategoryCmd.AddCommand(categoryPushCmd)
}
