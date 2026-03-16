package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/utils"
	"github.com/yarlson/pin"
)

var workflowPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull workflows from SuprSend workspace to local",
	Long:  `Pull workflows from SuprSend workspace to local`,
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		mode, _ := cmd.Flags().GetString("mode")
		outputDir, _ := cmd.Flags().GetString("dir")
		slug, _ := cmd.Flags().GetString("slug")
		force, _ := cmd.Flags().GetBool("force")
		if outputDir == "" {
			outputDir = filepath.Join(".", "suprsend", "workflow")
			if _, err := os.Stat(outputDir); os.IsNotExist(err) {
				if force {
					fmt.Fprintf(os.Stdout, "Using default directory: %s\n", outputDir)
				} else {
					outputDir = promptForOutputDirectory()
				}
			}
			if outputDir == "" {
				return fmt.Errorf("no output directory specified")
			}
		}
		if err := ensureOutputDirectory(outputDir); err != nil {
			fmt.Fprintf(os.Stdout, "Error with output directory: %v\n", err)
			return err
		}
		var p *pin.Pin
		if !utils.IsOutputPiped() {
			p = pin.New("Loading...",
				pin.WithSpinnerColor(pin.ColorCyan),
				pin.WithTextColor(pin.ColorYellow),
			)
			cancel := p.Start(context.Background())
			defer cancel()
		}

		mgmntClient := utils.GetSuprSendMgmntClient()
		if slug != "" {
			workflowResp, err := mgmntClient.GetWorkflowDetailBySlug(workspace, slug, mode)
			if err != nil {
				fmt.Fprintf(os.Stdout, "Error: Failed to get workflow detail: %v\n", err)
				return err
			}
			workflowJson, err := json.MarshalIndent(workflowResp, "", "  ")
			if err != nil {
				fmt.Fprintf(os.Stdout, "Error: Failed to marshal workflow: %v\n", err)
				return err
			}
			if err := os.WriteFile(filepath.Join(outputDir, fmt.Sprintf("%s.json", slug)), workflowJson, 0644); err != nil {
				fmt.Fprintf(os.Stdout, "Error: Failed to write workflow file: %v\n", err)
				return err
			}
			if p != nil {
				p.Stop(fmt.Sprintf("Pulled %s from %s", slug, workspace))
			}
			return nil
		}

		workflows_resp, err := mgmntClient.GetWorkflows(workspace, mode)
		if err != nil {
			fmt.Fprintf(os.Stdout, "Error: Failed to get workflows: %v\n", err)
			return err
		}
		if p != nil {
			p.Stop(fmt.Sprintf("Pulled %d workflows from %s", len(workflows_resp.Results), workspace))
		}

		stats, err := WriteWorkflowsToFiles(*workflows_resp, outputDir)
		if err != nil {
			fmt.Fprintf(os.Stdout, "Error: Failed to save workflows: %v\n", err)
			return err
		}

		fmt.Fprintf(os.Stdout, "\n=== Workflow Pull Summary ===\n")
		fmt.Fprintf(os.Stdout, "Total workflows processed: %d\n", stats.Total)
		fmt.Fprintf(os.Stdout, "Successfully updated: %d\n", stats.Success)
		fmt.Fprintf(os.Stdout, "Failed to pull: %d\n", stats.Failed)

		if stats.Failed > 0 {
			fmt.Fprintf(os.Stdout, "\nFailed workflows:\n")
			for _, errorMsg := range stats.Errors {
				fmt.Fprintf(os.Stdout, "  - %s\n", errorMsg)
			}
		}
		return nil
	},
}

func init() {
	workflowPullCmd.PersistentFlags().StringP("mode", "m", "live", "Mode of workflows to pull from (draft, live)")
	workflowPullCmd.PersistentFlags().StringP("dir", "d", "", "Output directory for workflows (default: ./suprsend/workflow)")
	workflowPullCmd.PersistentFlags().StringP("slug", "g", "", "Slug of the workflow to pull")
	workflowPullCmd.PersistentFlags().BoolP("force", "f", false, "Force using default directory without prompting")
	WorkflowCmd.AddCommand(workflowPullCmd)
}
