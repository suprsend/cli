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
	Categories   interface{}                                   `json:"categories"`
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
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		path, _ := cmd.Flags().GetString("dir")
		commit, _ := cmd.Flags().GetBool("commit")
		commitMessage, _ := cmd.Flags().GetString("commit-message")
		jsonPayload, _ := cmd.Flags().GetString("json")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		if jsonPayload != "" {
			var input jsonCategoryInput
			if err := json.Unmarshal([]byte(jsonPayload), &input); err != nil {
				return clierr.Wrap(err, clierr.CodeFileParseFailed, "")
			}
			if input.Categories == nil {
				return clierr.New("--json payload missing required \"categories\" field", clierr.CodeUnknown)
			}

			mgmntClient := utils.GetSuprSendMgmntClient()

			// Count pushable (non-English) translations up-front for the
			// dry-run summary; the API rejects English-locale pushes so
			// they're skipped on the real path too.
			pushableLocales := 0
			for locale := range input.Translations {
				if locale != "en" {
					pushableLocales++
				}
			}

			spinner := utils.NewSpinner("Pushing categories...")

			if dryRun {
				spinner.Stop(fmt.Sprintf("DRY RUN: would push categories and %d translation(s) to %s", pushableLocales, workspace))
				return nil
			}

			// Translations don't have a draft/live distinction — push them
			// regardless of --commit so they always reflect the local state.
			for locale, t := range input.Translations {
				if locale == "en" {
					continue
				}
				if err := mgmntClient.PushPreferenceTranslation(workspace, locale, t); err != nil {
					log.WithError(err).Errorf("Failed to push translation for locale %s", locale)
				}
			}

			if err := mgmntClient.PushCategories(workspace, input.Categories, commit, commitMessage); err != nil {
				log.WithError(err).Error("Couldn't push categories")
				return clierr.Wrap(err, clierr.CodeAPIInternal, "")
			}
			spinner.Stop(fmt.Sprintf("Pushed categories to %s", workspace))
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
		// draft/live model, so any local change should land immediately —
		// this also matches the docstring ("Upload local preference
		// categories and translations to a workspace"). Errors only when no
		// pushable locales exist (English-only) are downgraded to a debug
		// log so a translation-less workspace doesn't fail the category push.
		if err := translation.PushTranslations(workspace, "", translationDir, dryRun); err != nil {
			var ce *clierr.CLIError
			if errors.As(err, &ce) && ce.Code == clierr.CodeInvalidUsage {
				log.Debugf("No translations to push: %v", err)
			} else if errors.As(err, &ce) && ce.Code == clierr.CodeFileNotFound {
				log.Debugf("No translation files found: %v", err)
			} else {
				log.WithError(err).Warn("Translation push had errors; continuing with categories")
			}
		}

		spinner2 := utils.NewSpinner("Pushing categories...")

		if dryRun {
			spinner2.Stop(fmt.Sprintf("DRY RUN: would push categories to %s", workspace))
			return nil
		}

		mgmnt_client := utils.GetSuprSendMgmntClient()
		err = mgmnt_client.PushCategories(workspace, categories, commit, commitMessage)
		if err != nil {
			log.WithError(err).Error("Couldn't push categories")
			return clierr.Wrap(err, clierr.CodeAPIInternal, "")
		}
		spinner2.Stop(fmt.Sprintf("Pushed categories to %s", workspace))
		return nil
	},
}

func init() {
	categoryPushCmd.Flags().StringP("dir", "d", "", "Directory containing category files (default: ./"+defaultCategoryDir+")")
	categoryPushCmd.PersistentFlags().BoolP("commit", "c", false, "Promote changes from draft to live after pushing")
	categoryPushCmd.PersistentFlags().String("commit-message", "", "Message describing the changes being committed")
	categoryPushCmd.Flags().StringP("json", "j", "", `Categories (and optional translations) as a JSON object. Required "categories" key holds the preference category structure. Optional "translations" key maps locale codes to objects with "sections" and "categories" keys, e.g. '{"categories":{"root_categories":[...]},"translations":{"es":{"sections":{"key":{"name":"...","description":"..."}},"categories":{"key":{"name":"...","description":"..."}}}}}'`)
	categoryPushCmd.Flags().BoolP("dry-run", "n", false, "Print what would be pushed without making any changes")
	CategoryCmd.AddCommand(categoryPushCmd)
}
