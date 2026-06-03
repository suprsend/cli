/*
Copyright © 2025 SuprSend
*/
package workflow

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/utils"
)

var workflowListCmd = &cobra.Command{
	Use:   "list",
	Short: "List workflows for a workspace",
	Long:  `List workflows in a workspace with pagination. Returns workflow slug, name, status, and version info. Use --mode to switch between draft and live versions.`,
	Example: `  # List all workflows (live mode)
  suprsend workflow list

  # List draft workflows
  suprsend workflow list --mode draft

  # Paginate with JSON output
  suprsend workflow list --limit 50 --offset 50 --output json`,
	Annotations: map[string]string{
		"skills:tip:output": "Use `-o json` for machine-readable JSON output, `-o yaml` for YAML. Default `-o pretty` outputs a human-friendly table.",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		spinner := utils.NewSpinner("Loading...")
		workspace, _ := cmd.Flags().GetString("workspace")
		mgmntClient := utils.GetSuprSendMgmntClient()

		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")
		mode, _ := cmd.Flags().GetString("mode")

		workflows, err := mgmntClient.ListWorkflows(cmd.Context(), workspace, limit, offset, mode)
		if err != nil {
			log.WithError(err).Error("Couldn't fetch workflows")
			return clierr.Wrap(err, clierr.CodeAPIInternal, "")
		}

		spinner.Stop(fmt.Sprintf("Listed %d workflows from %s with offset %d", len(workflows.Results), workspace, offset))
		outputType, _ := cmd.Flags().GetString("output")
		if err := utils.ValidateOutputType(outputType, "pretty", "json", "yaml"); err != nil {
			return err
		}

		if len(workflows.Results) == 0 && utils.IsOutputPiped() {
			utils.OutputData([]any{}, outputType)
			return nil
		}
		utils.OutputData(workflows.Results, outputType)
		return nil
	},
}

func init() {
	workflowListCmd.PersistentFlags().IntP("limit", "l", 20, "Maximum number of workflows to return")
	workflowListCmd.PersistentFlags().Int("offset", 0, "Number of workflows to skip for pagination")
	workflowListCmd.PersistentFlags().StringP("mode", "m", "live", "Version mode: draft or live")
	workflowListCmd.PersistentFlags().StringP("output", "o", "pretty", "Output format: pretty, json, or yaml")
	workflowListCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		cmd.Parent().HelpFunc()(cmd, args)
	})
	WorkflowCmd.PersistentFlags().StringP("workspace", "w", "staging", "Workspace name (e.g., staging, production)")
	WorkflowCmd.AddCommand(workflowListCmd)
}
