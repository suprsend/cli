package workflow

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/utils"
)

var workflowCommitCmd = &cobra.Command{
	Use:   "commit [<slug>]",
	Short: "Commit workflow from draft to live",
	Long:  `Promote a workflow from draft to live mode. Pass the workflow slug as a positional argument or via --slug. Once committed, the workflow changes become active immediately.`,
	Example: `  # Commit a workflow to live (positional slug)
  suprsend workflow commit welcome

  # Commit using the flag form
  suprsend workflow commit --slug welcome

  # Dry run: see what would be committed without making changes
  suprsend workflow commit welcome --dry-run`,
	Annotations: map[string]string{
		"skills:tip.a-irreversible": "Commit is irreversible: it promotes the draft to **live**, and live workflows immediately begin executing the new definition for new trigger events.",
		"skills:tip.b-inspect":      "If you didn't author the draft locally, run `suprsend workflow get --slug <slug> --mode draft` first to inspect what will become live.",
	},
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		slug := utils.ResolveSlug(cmd, args)
		if slug == "" {
			return clierr.New("workflow slug is required: provide it as a positional argument or via --slug", clierr.CodeInvalidUsage)
		}

		workspace, _ := cmd.Flags().GetString("workspace")
		commitMessage, _ := cmd.Flags().GetString("commit-message")

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		report := utils.NewReport("workflow commit", workspace, dryRun)
		if dryRun {
			report.Add("workflows/"+slug, utils.StatusWouldCommit, "")
			return report.Finish(nil)
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
		commitStatus, err := mgmntClient.FinalizeWorkflow(workspace, slug, commitMessage)
		spinner.Stop("")
		if err != nil {
			err = clierr.Wrap(err, clierr.CodeAPIInternal, "")
			report.Fail("workflows/"+slug, err)
			return report.Finish(err)
		}
		status, detail := utils.PushStatus(commitStatus)
		report.Add("workflows/"+slug, status, detail)
		return report.Finish(nil)
	},
}

func init() {
	workflowCommitCmd.Flags().StringP("slug", "g", "", "Workflow slug")
	workflowCommitCmd.Flags().String("commit-message", "", "Message describing the changes being committed")
	workflowCommitCmd.Flags().BoolP("dry-run", "n", false, "Report what would be committed without making any changes")
	workflowCommitCmd.Flags().BoolP("force", "F", false, "Skip confirmation prompt")
	WorkflowCmd.AddCommand(workflowCommitCmd)
}
