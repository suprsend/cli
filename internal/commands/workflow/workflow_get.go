package workflow

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/utils"
	"github.com/yarlson/pin"
)

var workflowGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get workflow details",
	Long:  "Get workfow details of a specific wf. Example: suprsend workflow get --slug <slug>",
	RunE: func(cmd *cobra.Command, args []string) error {
		slug, _ := cmd.Flags().GetString("slug")
		if slug == "" {
			log.Error("Workflow slug is required. Example: suprsend workflow get --slug <slug>")
			return fmt.Errorf("workflow slug is required. Example: suprsend workflow get --slug <slug>")
		}
		workspace, _ := cmd.Flags().GetString("workspace")
		mode, _ := cmd.Flags().GetString("mode")
		outputType, _ := cmd.Flags().GetString("output")
		mgmntClient := utils.GetSuprSendMgmntClient()
		var p *pin.Pin
		var cancel context.CancelFunc
		if !utils.IsOutputPiped() {
			p = pin.New("Getting details...",
				pin.WithSpinnerColor(pin.ColorCyan),
				pin.WithTextColor(pin.ColorYellow),
			)
			cancel = p.Start(context.Background())
		}

		workflow, err := mgmntClient.GetWorkflowDetailBySlug(workspace, slug, mode)
		if err != nil {
			if p != nil {
				p.Stop("")
				cancel()
			}
			log.WithError(err).Errorf("Error getting workflow detail")
			return err
		}

		if p != nil {
			p.Stop(fmt.Sprintf("Successfully got details for '%s'", slug))
			cancel()
		}

		utils.OutputData(workflow, outputType)
		return nil
	},
}

func init() {
	workflowGetCmd.PersistentFlags().StringP("slug", "g", "", "Slug of the workflow to get")
	workflowGetCmd.PersistentFlags().String("mode", "live", "mode to fetch worklfow from.")
	workflowGetCmd.PersistentFlags().StringP("output", "o", "json", "Output format (json, yaml)")
	WorkflowCmd.AddCommand(workflowGetCmd)
}
