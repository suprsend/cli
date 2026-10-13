package category

import (
	"fmt"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/commands/category/translation"
	"github.com/suprsend/cli/internal/utils"
)

var categoryCommitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Commit categories",
	Long:  "Promote preference categories from draft to live mode. Before committing, it also pushes every non-English translation file from the `translations/` subdirectory of --dir (default `./suprsend/preference_categories/translations`); translation push errors are ignored and do not stop the commit.",
	Example: `  # Commit categories to live
  suprsend category commit

  # Commit in the production workspace
  suprsend category commit --workspace production

  # Commit with a message
  suprsend category commit --commit-message "Update notification preferences"`,
	Annotations: map[string]string{
		"skills:tip.a-irreversible": "Commit is irreversible: it promotes the draft to **live**. End-users immediately see the new preference categories on their notification settings page.",
		"skills:tip.b-translations": "Local translation files in the `translations/` subdirectory are pushed (live) before the commit. Translation push failures are only logged: the commit still runs and the command exits 0. Check the output for `failed to push` errors, or run `suprsend category translation push` first (it exits non-zero when a locale fails).",
		"skills:tip.c-confirm":      "The confirmation prompt only appears when stdin is a TTY. In non-interactive shells (CI, agents) the commit proceeds without asking, even without `-F`. Use `--dry-run` if you only want to check the command.",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		commitMsg, _ := cmd.Flags().GetString("commit-message")
		dir, _ := cmd.Flags().GetString("dir")

		// Determine the category directory
		categoryDir := dir
		if categoryDir == "" {
			categoryDir = filepath.Join(".", defaultCategoryDir)
		}

		translationDir := filepath.Join(categoryDir, "translations")

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		if dryRun {
			log.Infof("DRY RUN: would commit categories to %s", workspace)
			return nil
		}

		force, _ := cmd.Flags().GetBool("force")
		if !force {
			msg := fmt.Sprintf("This will promote categories to live in workspace \"%s\". Continue?", workspace)
			confirmed, err := utils.ConfirmDestructiveAction(msg)
			if err != nil || !confirmed {
				log.Info("Aborted.")
				return nil
			}
		}

		spinner := utils.NewSpinner("Loading...")

		_, _ = translation.PushTranslations(workspace, "", translationDir, false)

		mgmntClient := utils.GetSuprSendMgmntClient()
		err := mgmntClient.FinalizeCategories(workspace, commitMsg)
		if err != nil {
			log.WithError(err).Error("Couldn't commit categories")
			return clierr.Wrap(err, clierr.CodeAPIInternal, "")
		}
		spinner.Stop(fmt.Sprintf("Committed categories to %s", workspace))
		return nil
	},
}

func init() {
	categoryCommitCmd.Flags().StringP("dir", "d", "", "Directory containing category and translation files (default: ./"+defaultCategoryDir+")")
	categoryCommitCmd.PersistentFlags().String("commit-message", "", "Message describing the changes being committed")
	categoryCommitCmd.Flags().BoolP("dry-run", "n", false, "Print what would be committed without making any changes")
	categoryCommitCmd.Flags().BoolP("force", "F", false, "Skip confirmation prompt")
	CategoryCmd.AddCommand(categoryCommitCmd)
}
