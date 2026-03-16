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

var workflowGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get workflow details",
	Long:  "Get workfow details of a specific wf. Example: suprsend workflow get <slug>",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			log.Error("Workflow slug argument is required. Example: suprsend workflow get <slug>")
			return fmt.Errorf("Workflow slug argument is required. Example: suprsend workflow get <slug>")
		}
		slug := args[0]
		workspace, _ := cmd.Flags().GetString("workspace")
		mode, _ := cmd.Flags().GetString("mode")
		outputType, _ := cmd.Flags().GetString("output")
		mgmntClient := utils.GetSuprSendMgmntClient()
		var p *pin.Pin
		if !utils.IsOutputPiped() {
			p = pin.New("Getting details...",
				pin.WithSpinnerColor(pin.ColorCyan),
				pin.WithTextColor(pin.ColorYellow),
			)
			cancel := p.Start(context.Background())
			defer cancel()
		}

		workflow, err := mgmntClient.GetWorkflowDetail(workspace, slug, mode)
		if err != nil {
			log.WithError(err).Errorf("Error getting workflow detail")
			return err
		}
		utils.OutputData(workflow, outputType)
		if p != nil {
			p.Stop(fmt.Sprintf("Successfully got details for '%s'", slug))
		} else {
			fmt.Fprintf(os.Stdout, "Successfully got details for '%s'", slug)
		}
		return nil
	},
}

func init() {
	workflowGetCmd.PersistentFlags().String("mode", "live", "mode to fetch worklfow from.")
	WorkflowCmd.AddCommand(workflowGetCmd)
}
