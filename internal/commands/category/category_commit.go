package category

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/commands/category/translation"
	"github.com/suprsend/cli/internal/utils"
	"github.com/yarlson/pin"
)

var categoryCommitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Commit categories",
	Long:  "Promote preference categories from draft to live mode. Also pushes any local translation files from the translations subdirectory before committing.",
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
			fmt.Fprintf(os.Stdout, "DRY RUN: would commit categories to %s\n", workspace)
			return nil
		}

		force, _ := cmd.Flags().GetBool("force")
		if !force {
			msg := fmt.Sprintf("This will promote categories to live in workspace \"%s\". Continue?", workspace)
			confirmed, err := utils.ConfirmDestructiveAction(msg)
			if err != nil || !confirmed {
				fmt.Fprintln(os.Stdout, "Aborted.")
				return nil
			}
		}

		var p *pin.Pin
		if !utils.IsOutputPiped() {
			p = pin.New("Loading...",
				pin.WithSpinnerColor(pin.ColorCyan),
				pin.WithTextColor(pin.ColorYellow),
			)
			cancel := p.Start(context.Background())
			defer cancel()
		}

		translation.PushTranslations(workspace, "", translationDir)

		mgmntClient := utils.GetSuprSendMgmntClient()
		err := mgmntClient.FinalizeCategories(workspace, commitMsg)
		if err != nil {
			log.WithError(err).Error("Couldn't commit categories")
			return err
		}
		if p != nil {
			p.Stop(fmt.Sprintf("Committed categories to %s", workspace))
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
