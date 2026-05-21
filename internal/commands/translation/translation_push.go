package translation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/utils"
)

var translationPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push translation files to a workspace",
	Long:  "Upload local template translation JSON files to a workspace. Reads all .json files from the input directory and pushes them. Use --commit to also finalize the changes immediately.",
	Example: `  # Push translations from default directory
  suprsend translation push

  # Push and commit to live immediately
  suprsend translation push --commit

  # Dry run: preview what would be pushed without making changes
  suprsend translation push --dry-run`,
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		outputDir, _ := cmd.Flags().GetString("dir")
		commit, _ := cmd.Flags().GetBool("commit")
		commitMessage, _ := cmd.Flags().GetString("commit-message")
		jsonPayload, _ := cmd.Flags().GetString("json")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		force, _ := cmd.Flags().GetBool("force")
		var dryRunNames []string

		if commit && !dryRun && !force {
			msg := fmt.Sprintf("This will push and promote translations to live in workspace \"%s\". Continue?", workspace)
			confirmed, err := utils.ConfirmDestructiveAction(msg)
			if err != nil || !confirmed {
				log.Info("Aborted.")
				return nil
			}
		}

		mgmntClient := utils.GetSuprSendMgmntClient()

		hasError := false
		var spinner *utils.Spinner
		stats := &TranslationPushStats{
			Errors: []string{},
		}

		if jsonPayload != "" {
			// Parse as map of filename -> content
			var translations map[string]map[string]any
			if err := json.Unmarshal([]byte(jsonPayload), &translations); err != nil {
				return clierr.Wrap(err, clierr.CodeFileParseFailed, "")
			}

			for filename, content := range translations {
				stats.Total++
				if !hasError {
					spinner = utils.NewSpinner(fmt.Sprintf("Pushing %s.json...", filename))
				}

				if dryRun {
					dryRunNames = append(dryRunNames, filename+".json")
					stats.Success++
					spinner.Stop(fmt.Sprintf("(dry run) %s.json", filename))
					hasError = false
					continue
				}

				err := mgmntClient.PushTranslation(workspace, filename+".json", map[string]any{"content": content})
				if err != nil {
					spinner.Stop("")
					hasError = true
					log.Errorf("translations/%s.json: failed to push: %v", filename, err)
					stats.Failed++
					stats.Errors = append(stats.Errors, fmt.Sprintf("translations/%s.json: failed to push: %v", filename, err))
					continue
				}

				stats.Success++
				spinner.Stop(fmt.Sprintf("Pushed translation: %s.json", filename))
				hasError = false
			}
		} else {
			if outputDir == "" {
				outputDir = filepath.Join(".", "suprsend", "translations")
			}

			files, err := os.ReadDir(outputDir)
			if err != nil {
				log.WithError(err).Errorf("Failed to read local translation directory")
				return clierr.Wrap(err, clierr.CodeFileNotFound, "")
			}

			log.Infof("Pushing translations to %s", workspace)

			for _, file := range files {
				if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
					continue
				}

				if !hasError {
					spinner = utils.NewSpinner(fmt.Sprintf("Pushing %s...", file.Name()))
				}

				stats.Total++
				path := filepath.Join(outputDir, file.Name())
				data, err := os.ReadFile(path)
				if err != nil {
					spinner.Stop("")
					hasError = true
					log.Errorf("Failed to read file %s: %v", file.Name(), err)
					stats.Failed++
					stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to read file %s: %v", file.Name(), err))
					continue
				}

				var content map[string]any
				if err := json.Unmarshal(data, &content); err != nil {
					spinner.Stop("")
					hasError = true
					log.Errorf("Failed to parse JSON for %s: %v", file.Name(), err)
					stats.Failed++
					stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to parse JSON for %s: %v", file.Name(), err))
					continue
				}

				if dryRun {
					dryRunNames = append(dryRunNames, file.Name())
					stats.Success++
					spinner.Stop(fmt.Sprintf("(dry run) %s", file.Name()))
					hasError = false
					continue
				}

				err = mgmntClient.PushTranslation(workspace, file.Name(), map[string]any{"content": content})
				if err != nil {
					spinner.Stop("")
					hasError = true
					log.Errorf("translations/%s: failed to push: %v", file.Name(), err)
					stats.Failed++
					stats.Errors = append(stats.Errors, fmt.Sprintf("translations/%s: failed to push: %v", file.Name(), err))
					continue
				}

				stats.Success++
				spinner.Stop(fmt.Sprintf("Pushed translation: %s", file.Name()))
				hasError = false
			}
		}

		if dryRun {
			action := "push"
			if commit {
				action = "push and commit"
			}
			log.Infof("DRY RUN: would %s %d translation(s) to %s", action, len(dryRunNames), workspace)
			for _, n := range dryRunNames {
				log.Infof("  - %s", n)
			}
			return nil
		}

		log.Info("=== Translation Push Summary ===")
		log.Infof("Total translations processed: %d", stats.Total)
		log.Infof("Successfully pushed: %d", stats.Success)
		log.Infof("Failed to push: %d", stats.Failed)

		if stats.Failed > 0 {
			log.Info("Failed translations:")
			for _, errMsg := range stats.Errors {
				log.Infof("  - %s", errMsg)
			}
		}

		if commit {
			if err := mgmntClient.FinalizeTranslation(workspace, commitMessage); err != nil {
				log.Errorf("Failed to commit translation: %v", err)
				return clierr.Wrap(err, clierr.CodeAPIInternal, "")
			}
			log.Infof("Committed translation: %s", commitMessage)
		}

		if stats.Failed > 0 {
			return fmt.Errorf("%d translation(s) failed to push", stats.Failed)
		}
		return nil
	},
}

func init() {
	translationPushCmd.Flags().BoolP("commit", "c", false, "Promote changes from draft to live after pushing")
	translationPushCmd.Flags().String("commit-message", "", "Message describing the changes being committed")
	translationPushCmd.Flags().StringP("dir", "d", "", "Directory containing translation JSON files (default: ./suprsend/translations)")
	translationPushCmd.Flags().StringP("json", "j", "", `Translations as a JSON object mapping locale codes (without .json extension) to their translation content objects, e.g. '{"en":{"key":"value"},"fr":{"key":"valeur"}}'`)
	translationPushCmd.Flags().BoolP("dry-run", "n", false, "Print what would be pushed without making any changes")
	translationPushCmd.Flags().BoolP("force", "F", false, "Skip confirmation prompt when --commit is set")
	TranslationCmd.AddCommand(translationPushCmd)
}
