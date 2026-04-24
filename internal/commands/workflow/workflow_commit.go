package workflow

import (
	"context"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/utils"
	"github.com/yarlson/pin"
)

var workflowCommitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Commit workflow from draft to live",
	Long:  `Promote a workflow from draft to live mode. Requires a workflow slug as a positional argument. Once committed, the workflow changes become active immediately.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			log.Error("Workflow slug argument is required. Example: suprsend workflow commit <slug>")
			return fmt.Errorf("workflow slug argument is required. Example: suprsend workflow commit <slug>")
		}
		slug := args[0]

		workspace, _ := cmd.Flags().GetString("workspace")
		commitMessage, _ := cmd.Flags().GetString("commit-message")

		mgmntClient := utils.GetSuprSendMgmntClient()
		var p *pin.Pin
		if !utils.IsOutputPiped() {
			p = pin.New("Committing workflow...",
				pin.WithSpinnerColor(pin.ColorCyan),
				pin.WithTextColor(pin.ColorYellow),
			)
			cancel := p.Start(context.Background())
			defer cancel()
		}

		err := mgmntClient.FinalizeWorkflow(workspace, slug, commitMessage)
		if err != nil {
			log.Error(err.Error())
			return err
		}

		if p != nil {
			p.Stop(fmt.Sprintf("Successfully committed workflow '%s' to live mode", slug))
		} else {
			fmt.Fprintf(os.Stdout, "Successfully committed workflow '%s' to live mode\n", slug)
		}
		return nil
	},
}

func init() {
	workflowCommitCmd.Flags().String("commit-message", "", "Message describing the changes being committed")
	WorkflowCmd.AddCommand(workflowCommitCmd)
}
