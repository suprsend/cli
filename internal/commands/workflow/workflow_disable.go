package workflow

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/utils"
)

var workflowDisableCmd = &cobra.Command{
	Use:   "disable [<slug>]",
	Short: "Disable a workflow",
	Long:  "Disable a workflow to stop it from processing triggers. Pass the workflow slug as a positional argument or via --slug.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")

		slug := utils.ResolveSlug(cmd, args)
		if slug == "" {
			log.Error("workflow slug is required: provide it as a positional argument or via --slug")
			return fmt.Errorf("workflow slug is required: provide it as a positional argument or via --slug")
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		if dryRun {
			fmt.Printf("DRY RUN: would disable workflow '%s' in %s\n", slug, workspace)
			return nil
		}

		force, _ := cmd.Flags().GetBool("force")
		if !force {
			msg := fmt.Sprintf("This will disable workflow '%s' in workspace \"%s\". Live traffic for this workflow will stop. Continue?", slug, workspace)
			confirmed, err := utils.ConfirmDestructiveAction(msg)
			if err != nil || !confirmed {
				fmt.Println("Aborted.")
				return nil
			}
		}

		mgmntClient := utils.GetSuprSendMgmntClient()

		err := mgmntClient.ChangeStatusWorkflow(workspace, slug, false)
		if err != nil {
			log.Error(err.Error())
			return err
		}

		fmt.Printf("Disabled workflow: %s\n", slug)
		return nil
	},
}

func init() {
	workflowDisableCmd.Flags().StringP("slug", "g", "", "Workflow slug")
	workflowDisableCmd.Flags().BoolP("dry-run", "n", false, "Print what would be changed without making any changes")
	workflowDisableCmd.Flags().BoolP("force", "F", false, "Skip confirmation prompt")
	WorkflowCmd.AddCommand(workflowDisableCmd)
}
