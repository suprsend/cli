package workflow

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/utils"
)

var workflowDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable a workflow",
	Long:  "Disable a workflow to deactivate. Example: suprsend workflow disable <slug>",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			log.Error("workflow_slug is required.")
			return fmt.Errorf("workflow_slug is required.")
		}
		workspace, _ := cmd.Flags().GetString("workspace")

		slug := args[0]

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
	WorkflowCmd.AddCommand(workflowDisableCmd)
}
