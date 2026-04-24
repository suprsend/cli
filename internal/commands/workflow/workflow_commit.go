package workflow

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/utils"
)

var workflowCommitCmd = &cobra.Command{
	Use:   "commit [<slug>]",
	Short: "Commit workflow from draft to live",
	Long:  `Promote a workflow from draft to live mode. Pass the workflow slug as a positional argument or via --slug. Once committed, the workflow changes become active immediately.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		slug := utils.ResolveSlug(cmd, args)
		if slug == "" {
			log.Error("workflow slug is required: provide it as a positional argument or via --slug")
			return fmt.Errorf("workflow slug is required: provide it as a positional argument or via --slug")
		}

		workspace, _ := cmd.Flags().GetString("workspace")
		commitMessage, _ := cmd.Flags().GetString("commit-message")

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		if dryRun {
			log.Infof("DRY RUN: would commit workflow '%s' to live", slug)
			return nil
		}

		force, _ := cmd.Flags().GetBool("force")
		if !force {
			msg := fmt.Sprintf("This will promote workflow '%s' to live in workspace \"%s\". Continue?", slug, workspace)
			confirmed, err := utils.ConfirmDestructiveAction(msg)
			if err != nil || !confirmed {
				log.Info("Aborted.")
				return nil
			}
		}

		mgmntClient := utils.GetSuprSendMgmntClient()
		spinner := utils.NewSpinner("Committing workflow...")

		err := mgmntClient.FinalizeWorkflow(workspace, slug, commitMessage)
		if err != nil {
			log.Error(err.Error())
			return err
		}

		spinner.Stop(fmt.Sprintf("Successfully committed workflow '%s' to live mode", slug))
		return nil
	},
}

func init() {
	workflowCommitCmd.Flags().StringP("slug", "g", "", "Workflow slug")
	workflowCommitCmd.Flags().String("commit-message", "", "Message describing the changes being committed")
	workflowCommitCmd.Flags().BoolP("dry-run", "n", false, "Print what would be committed without making any changes")
	workflowCommitCmd.Flags().BoolP("force", "F", false, "Skip confirmation prompt")
	WorkflowCmd.AddCommand(workflowCommitCmd)
}
