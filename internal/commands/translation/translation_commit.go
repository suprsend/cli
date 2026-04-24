package translation

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/utils"
	"github.com/yarlson/pin"
)

var translationCommitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Commit translation",
	Long:  "Promote template translation changes from draft to live mode. Finalizes all pending translation changes in the workspace.",
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		commitMessage, _ := cmd.Flags().GetString("commit-message")

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		if dryRun {
			log.Infof("DRY RUN: would commit translations in %s", workspace)
			return nil
		}

		force, _ := cmd.Flags().GetBool("force")
		if !force {
			msg := fmt.Sprintf("This will promote translations to live in workspace \"%s\". Continue?", workspace)
			confirmed, err := utils.ConfirmDestructiveAction(msg)
			if err != nil || !confirmed {
				log.Info("Aborted.")
				return nil
			}
		}

		mgmntClient := utils.GetSuprSendMgmntClient()
		var p *pin.Pin
		if !utils.IsOutputPiped() {
			p = pin.New("Committing translation...",
				pin.WithSpinnerColor(pin.ColorCyan),
				pin.WithTextColor(pin.ColorYellow),
			)
			cancel := p.Start(context.Background())
			defer cancel()
		}
		err := mgmntClient.FinalizeTranslation(workspace, commitMessage)
		if err != nil {
			log.Errorf("%s", err)
			return err
		}
		if p != nil {
			p.Stop(fmt.Sprintf("Successfully committed translation '%s'", commitMessage))
		} else {
			log.Infof("Successfully committed translation '%s'", commitMessage)
		}
		return nil
	},
}

func init() {
	translationCommitCmd.Flags().String("commit-message", "", "Message describing the changes being committed")
	translationCommitCmd.Flags().BoolP("dry-run", "n", false, "Print what would be committed without making any changes")
	translationCommitCmd.Flags().BoolP("force", "F", false, "Skip confirmation prompt")
	TranslationCmd.AddCommand(translationCommitCmd)
}
