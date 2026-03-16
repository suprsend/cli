/*
Copyright © 2025 SuprSend
*/
package workflow

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/utils"
	"github.com/yarlson/pin"
)

var workflowListCmd = &cobra.Command{
	Use:   "list",
	Short: "List workflows for a workspace",
	Long:  `List workflows for a workspace`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var p *pin.Pin
		if !utils.IsOutputPiped() {
			p = pin.New("Loading...",
				pin.WithSpinnerColor(pin.ColorCyan),
				pin.WithTextColor(pin.ColorYellow),
			)
			cancel := p.Start(context.Background())
			defer cancel()
		}
		workspace, _ := cmd.Flags().GetString("workspace")
		mgmntClient := utils.GetSuprSendMgmntClient()

		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")
		mode, _ := cmd.Flags().GetString("mode")

		workflows, err := mgmntClient.ListWorkflows(workspace, limit, offset, mode)
		if err != nil {
			log.WithError(err).Error("Couldn't fetch workflows")
			return err
		}

		msg := fmt.Sprintf("Listed %d workflows from %s with offset %d", len(workflows.Results), workspace, offset)
		if p != nil {
			p.Stop(msg)
		}
		outputType, _ := cmd.Flags().GetString("output")

		if len(workflows.Results) == 0 && utils.IsOutputPiped() {
			utils.OutputData([]interface{}{}, outputType)
			return nil
		}
		utils.OutputData(workflows.Results, outputType)
		return nil
	},
}

func init() {
	workflowListCmd.PersistentFlags().IntP("limit", "l", 20, "Limit the number of workflows to list")
	workflowListCmd.PersistentFlags().IntP("offset", "f", 0, "Offset the number of workflows to list (default: 0)")
	workflowListCmd.PersistentFlags().StringP("mode", "m", "live", "Mode of workflows to list (draft, live), default: live")
	workflowListCmd.PersistentFlags().StringP("output", "o", "pretty", "Output Style (pretty, yaml, json)")
	workflowListCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		cmd.Parent().HelpFunc()(cmd, args)
	})
	WorkflowCmd.PersistentFlags().StringP("workspace", "w", "staging", "Workspace to list workflows from")
	WorkflowCmd.PersistentFlags().StringP("service-token", "s", "", "Service token (default: $SUPRSEND_SERVICE_TOKEN)")
	WorkflowCmd.AddCommand(workflowListCmd)
}
