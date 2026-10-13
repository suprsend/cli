package category

import (
	"errors"
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
	Long:  "Promote preference categories from draft to live mode. Before committing, it also pushes every non-English translation file from the `translations/` subdirectory of --dir (default `./suprsend/preference_categories/translations`); a translation push error does not stop the commit, but the command then exits non-zero.",
	Example: `  # Commit categories to live
  suprsend category commit

  # Commit in the production workspace
  suprsend category commit --workspace production

  # Commit with a message
  suprsend category commit --commit-message "Update notification preferences"`,
	Annotations: map[string]string{
		"skills:tip.a-irreversible": "Commit is irreversible: it promotes the draft to **live**. End-users immediately see the new preference categories on their notification settings page.",
		"skills:tip.b-translations": "Local translation files in the `translations/` subdirectory are pushed (live) before the commit. If a locale fails to push, the categories are still committed, but the command exits non-zero and names the failure. Run `suprsend category translation push` first to fix translations separately.",
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

		// Translations are not part of the commit (they have no draft state),
		// so a failed locale doesn't block it, but it does fail the command.
		_, terr := translation.PushTranslations(workspace, "", translationDir, false)
		var ce *clierr.CLIError
		if errors.As(terr, &ce) && (ce.Code == clierr.CodeInvalidUsage || ce.Code == clierr.CodeFileNotFound) {
			log.Debugf("No translations to push: %v", terr)
			terr = nil
		}

		mgmntClient := utils.GetSuprSendMgmntClient()
		err := mgmntClient.FinalizeCategories(workspace, commitMsg)
		if err != nil {
			log.WithError(err).Error("Couldn't commit categories")
			return clierr.Wrap(err, clierr.CodeAPIInternal, "")
		}
		spinner.Stop(fmt.Sprintf("Categories in %s are up to date in live", workspace))
		if terr != nil {
			return clierr.New(fmt.Sprintf("categories committed to %s, but translations failed to push: %v", workspace, terr), clierr.CodeAPIInternal)
		}
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
