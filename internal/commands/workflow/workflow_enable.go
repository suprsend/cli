package workflow

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/utils"
)

var worklowEnableCmd = &cobra.Command{
	Use:   "enable [<slug>]",
	Short: "Enables a workflow.",
	Long:  "Enable a workflow to make it active and ready to receive triggers. Pass the workflow slug as a positional argument or via --slug.",
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
			log.Infof("DRY RUN: would enable workflow '%s' in %s", slug, workspace)
			return nil
		}

		mgmntClient := utils.GetSuprSendMgmntClient()
		err := mgmntClient.ChangeStatusWorkflow(workspace, slug, true)
		if err != nil {
			log.Error(err.Error())
			return clierr.Wrap(err, clierr.CodeAPIInternal, "")
		}

		log.Infof("Enabled workflow: %s", slug)
		return nil
	},
}

func init() {
	worklowEnableCmd.Flags().StringP("slug", "g", "", "Workflow slug")
	worklowEnableCmd.Flags().BoolP("dry-run", "n", false, "Print what would be changed without making any changes")
	WorkflowCmd.AddCommand(worklowEnableCmd)
}
