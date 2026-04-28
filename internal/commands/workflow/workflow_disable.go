package workflow

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/utils"
)


var workflowDisableCmd = &cobra.Command{
	Use:   "disable [<slug>]",
	Short: "Disable a workflow",
	Long:  "Disable a workflow to stop it from processing triggers. Pass the workflow slug as a positional argument or via --slug.",
	Example: `  # Disable a workflow (prompts for confirmation)
  suprsend workflow disable welcome

  # Disable without confirmation prompt
  suprsend workflow disable welcome --force

  # Dry run: see what would change without making changes
  suprsend workflow disable welcome --dry-run`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")

		slug := utils.ResolveSlug(cmd, args)
		if slug == "" {
			return clierr.New("workflow slug is required: provide it as a positional argument or via --slug", clierr.CodeInvalidUsage)
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		if dryRun {
			log.Infof("DRY RUN: would disable workflow '%s' in %s", slug, workspace)
			return nil
		}

		force, _ := cmd.Flags().GetBool("force")
		if !force {
			msg := fmt.Sprintf("This will disable workflow '%s' in workspace \"%s\". Live traffic for this workflow will stop. Continue?", slug, workspace)
			confirmed, err := utils.ConfirmDestructiveAction(msg)
			if err != nil || !confirmed {
				log.Info("Aborted.")
				return nil
			}
		}

		mgmntClient := utils.GetSuprSendMgmntClient()

		err := mgmntClient.ChangeStatusWorkflow(workspace, slug, false)
		if err != nil {
			log.Error(err.Error())
			return clierr.Wrap(err, clierr.CodeAPIInternal, "")
		}

		log.Infof("Disabled workflow: %s", slug)
		return nil
	},
}

func init() {
	workflowDisableCmd.Flags().StringP("slug", "g", "", "Workflow slug")
	workflowDisableCmd.Flags().BoolP("dry-run", "n", false, "Print what would be changed without making any changes")
	workflowDisableCmd.Flags().BoolP("force", "F", false, "Skip confirmation prompt")
	WorkflowCmd.AddCommand(workflowDisableCmd)
}
