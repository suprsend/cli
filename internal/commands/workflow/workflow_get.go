package workflow

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/utils"
)

var workflowGetCmd = &cobra.Command{
	Use:   "get [<slug>]",
	Short: "Get workflow details",
	Long:  "Retrieve detailed information for a specific workflow by its slug. Returns the full workflow definition including nodes, connections, and configuration. Use --mode to switch between draft and live versions.",
	Example: `  # Get a workflow by slug (positional)
  suprsend workflow get welcome

  # Get using the flag form
  suprsend workflow get --slug welcome

  # Get the draft version
  suprsend workflow get welcome --mode draft`,
	Args: cobra.MaximumNArgs(1),
	Annotations: map[string]string{
		"skills:tip:output": "Use `-o json` for machine-readable JSON output, `-o yaml` for YAML. Default `-o pretty` outputs a human-friendly table.",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		slug := utils.ResolveSlug(cmd, args)
		if slug == "" {
			return clierr.New("slug is required: provide it as a positional argument or via --slug", clierr.CodeInvalidUsage)
		}
		workspace, _ := cmd.Flags().GetString("workspace")
		mode, _ := cmd.Flags().GetString("mode")
		outputType, _ := cmd.Flags().GetString("output")
		if err := utils.ValidateOutputType(outputType, "json", "yaml"); err != nil {
			return err
		}
		mgmntClient := utils.GetSuprSendMgmntClient()
		spinner := utils.NewSpinner("Getting details...")

		workflow, err := mgmntClient.GetWorkflowDetailBySlug(cmd.Context(), workspace, slug, mode)
		if err != nil {
			spinner.Stop("")
			log.WithError(err).Errorf("Error getting workflow detail")
			return clierr.Wrap(err, clierr.CodeAPIInternal, "")
		}

		spinner.Stop(fmt.Sprintf("Successfully got details for '%s'", slug))

		utils.OutputData(workflow, outputType)
		return nil
	},
}

func init() {
	workflowGetCmd.PersistentFlags().StringP("slug", "g", "", "Workflow slug")
	workflowGetCmd.PersistentFlags().StringP("mode", "m", "live", "Version mode: draft or live")
	workflowGetCmd.PersistentFlags().StringP("output", "o", "json", "Output format: json or yaml")
	WorkflowCmd.AddCommand(workflowGetCmd)
}
