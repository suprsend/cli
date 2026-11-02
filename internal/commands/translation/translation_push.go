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

		if commit && !dryRun && !force {
			msg := fmt.Sprintf("This will push and promote translations to live in workspace \"%s\". Continue?", workspace)
			confirmed, err := utils.ConfirmDestructiveAction(msg)
			if err != nil || !confirmed {
				log.Info("Aborted.")
				return nil
			}
		}

		mgmntClient := utils.GetSuprSendMgmntClient()
		report := utils.NewReport("translation push", workspace, dryRun)
		var failures []string
		fail := func(resource, action string, err error) {
			report.Fail(resource, err)
			failures = append(failures, clierr.ItemFailure(resource, action, err))
		}
		push := func(fileName string, content map[string]any) {
			resource := "translations/" + fileName
			if dryRun {
				report.Add(resource, utils.StatusWouldPush, "")
				return
			}
			spinner := utils.NewSpinner(fmt.Sprintf("Pushing %s...", fileName))
			err := mgmntClient.PushTranslation(workspace, fileName, map[string]any{"content": content})
			spinner.Stop("")
			if err != nil {
				fail(resource, "failed to push", err)
				return
			}
			report.Add(resource, utils.StatusPushed, "")
		}

		if jsonPayload != "" {
			// Parse as map of filename -> content
			var translations map[string]map[string]any
			if err := json.Unmarshal([]byte(jsonPayload), &translations); err != nil {
				return clierr.Wrap(err, clierr.CodeFileParseFailed, "")
			}
			for filename, content := range translations {
				push(filename+".json", content)
			}
		} else {
			if outputDir == "" {
				outputDir = filepath.Join(".", "suprsend", "translations")
			}

			files, err := os.ReadDir(outputDir)
			if err != nil {
				return clierr.Wrap(err, clierr.CodeFileNotFound, "")
			}

			for _, file := range files {
				if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
					continue
				}
				resource := "translations/" + file.Name()
				data, err := os.ReadFile(filepath.Join(outputDir, file.Name()))
				if err != nil {
					fail(resource, "failed to read", err)
					continue
				}
				var content map[string]any
				if err := json.Unmarshal(data, &content); err != nil {
					fail(resource, "failed to parse", err)
					continue
				}
				push(file.Name(), content)
			}
		}

		if dryRun {
			if commit {
				report.Add("translations", utils.StatusWouldCommit, "all pushed translations")
			}
			if len(failures) > 0 {
				return report.Finish(clierr.Failures(fmt.Sprintf("%d translation(s) failed local checks", len(failures)), failures, clierr.CodeFileParseFailed))
			}
			return report.Finish(nil)
		}

		if commit {
			spinner := utils.NewSpinner("Committing translations...")
			commitStatus, err := mgmntClient.FinalizeTranslation(workspace, commitMessage)
			spinner.Stop("")
			if err != nil {
				err = clierr.Wrap(err, clierr.CodeAPIInternal, "")
				report.Fail("translations", err)
				return report.Finish(err)
			}
			status, detail := utils.PushStatus(commitStatus)
			report.Add("translations", status, detail)
		}

		if len(failures) > 0 {
			return report.Finish(clierr.Failures(fmt.Sprintf("%d translation(s) failed to push", len(failures)), failures, clierr.CodeAPIInternal))
		}
		return report.Finish(nil)
	},
}

func init() {
	translationPushCmd.Flags().BoolP("commit", "c", false, "Promote changes from draft to live after pushing")
	translationPushCmd.Flags().String("commit-message", "", "Message describing the changes being committed")
	translationPushCmd.Flags().StringP("dir", "d", "", "Directory containing translation JSON files (default: ./suprsend/translations)")
	translationPushCmd.Flags().StringP("json", "j", "", `Translations as a JSON object mapping locale codes (without .json extension) to their translation content objects, e.g. '{"en":{"key":"value"},"fr":{"key":"valeur"}}'`)
	translationPushCmd.Flags().BoolP("dry-run", "n", false, "Report what would be pushed without making any changes")
	translationPushCmd.Flags().BoolP("force", "F", false, "Skip confirmation prompt when --commit is set")
	TranslationCmd.AddCommand(translationPushCmd)
}
