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
	Use:   "get [<slug>]",
	Short: "Get workflow details",
	Long:  "Retrieve detailed information for a specific workflow by its slug. Returns the full workflow definition including nodes, connections, and configuration. Use --mode to switch between draft and live versions.",
	Args:  cobra.MaximumNArgs(1),
	Annotations: map[string]string{
		"skills:tip:output": "Use `-o json` for machine-readable JSON output, `-o yaml` for YAML. Default `-o pretty` outputs a human-friendly table.",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		slugFlag, _ := cmd.Flags().GetString("slug")
		if len(args) > 0 && slugFlag != "" {
			return fmt.Errorf("slug provided both as positional argument and --slug flag; use only one")
		}
		var slug string
		if len(args) > 0 {
			slug = args[0]
		} else {
			slug = slugFlag
		}
		if slug == "" {
			return fmt.Errorf("slug is required: provide it as a positional argument or via --slug")
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
	workflowGetCmd.PersistentFlags().StringP("slug", "g", "", "Workflow slug")
	workflowGetCmd.PersistentFlags().StringP("mode", "m", "live", "Version mode: draft or live")
	workflowGetCmd.PersistentFlags().StringP("output", "o", "json", "Output format: json or yaml")
	WorkflowCmd.AddCommand(workflowGetCmd)
}
