package category

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/commands/category/translation"
	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/cli/mgmnt"
	"github.com/yarlson/pin"
)

type jsonCategoryInput struct {
	Categories   interface{}                                   `json:"categories"`
	Translations map[string]mgmnt.PreferenceTranslationContent `json:"translations"`
}

var categoryPushCmd = &cobra.Command{
	Use:   "push",
	Long: `Upload local preference categories and translations to a workspace. Reads categories_preferences.json and translation files from the input directory. By default, changes are committed immediately (--commit=true).

Examples:
  # Push from local files (default)
  suprsend category --workspace <workspace> push

  # Push from a custom directory
  suprsend category --workspace <workspace> push --dir ./my-dir

  # Push categories inline via JSON
  suprsend category --workspace <workspace> push --json '{"categories": {...}}'

  # Push categories + translations inline via JSON
  suprsend category --workspace <workspace> push --json '{"categories": {...}, "translations": {"es": {...}}}'`,
	Short: "Push categories to a workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		path, _ := cmd.Flags().GetString("dir")
		commit, _ := cmd.Flags().GetString("commit")
		commitMessage, _ := cmd.Flags().GetString("commit-message")
		jsonPayload, _ := cmd.Flags().GetString("json")

		if jsonPayload != "" {
			var input jsonCategoryInput
			if err := json.Unmarshal([]byte(jsonPayload), &input); err != nil {
				return fmt.Errorf("failed to parse --json payload: %w", err)
			}
			if input.Categories == nil {
				return fmt.Errorf("--json payload missing required \"categories\" field")
			}

			mgmntClient := utils.GetSuprSendMgmntClient()

			var p *pin.Pin
			if !utils.IsOutputPiped() {
				p = pin.New("Pushing categories...",
					pin.WithSpinnerColor(pin.ColorCyan),
					pin.WithTextColor(pin.ColorYellow),
				)
				cancel := p.Start(context.Background())
				defer cancel()
			}

			if commit == "true" {
				for locale, t := range input.Translations {
					if locale == "en" {
						continue
					}
					if err := mgmntClient.PushPreferenceTranslation(workspace, locale, t); err != nil {
						log.WithError(err).Errorf("Failed to push translation for locale %s", locale)
					}
				}
			}

			if err := mgmntClient.PushCategories(workspace, input.Categories, commit, commitMessage); err != nil {
				log.WithError(err).Error("Couldn't push categories")
				return err
			}
			if p != nil {
				p.Stop(fmt.Sprintf("Pushed categories to %s", workspace))
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
			return err
		}

		categories, err := ReadFromFile(path)
		if err != nil {
			log.WithError(err).Error("Couldn't read categories from file")
			return err
		}

		var p *pin.Pin
		if !utils.IsOutputPiped() {
			p = pin.New("Pushing categories...",
				pin.WithSpinnerColor(pin.ColorCyan),
				pin.WithTextColor(pin.ColorYellow),
			)
			cancel := p.Start(context.Background())
			defer cancel()
		}

		if commit == "true" {
			translation.PushTranslations(workspace, "", translationDir)
		}

		mgmnt_client := utils.GetSuprSendMgmntClient()
		err = mgmnt_client.PushCategories(workspace, categories, commit, commitMessage)
		if err != nil {
			log.WithError(err).Error("Couldn't push categories")
			return err
		}
		if p != nil {
			p.Stop(fmt.Sprintf("Pushed categories to %s", workspace))
		}
		return nil
	},
}

func init() {
	categoryPushCmd.Flags().StringP("dir", "d", "", "Directory containing category files (default: ./"+defaultCategoryDir+")")
	categoryPushCmd.PersistentFlags().StringP("commit", "c", "true", "Promote changes from draft to live after pushing (true/false)")
	categoryPushCmd.PersistentFlags().StringP("commit-message", "m", "", "Message describing the changes being committed")
	categoryPushCmd.Flags().StringP("json", "j", "", `Categories (and optional translations) as a JSON object. Required "categories" key holds the preference category structure. Optional "translations" key maps locale codes to objects with "sections" and "categories" keys, e.g. '{"categories":{"root_categories":[...]},"translations":{"es":{"sections":{"key":{"name":"...","description":"..."}},"categories":{"key":{"name":"...","description":"..."}}}}}'`)
	CategoryCmd.AddCommand(categoryPushCmd)
}
