package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/utils"
)

var workflowPullCmd = &cobra.Command{
	Use:   "pull [<slug>]",
	Short: "Pull workflows from SuprSend workspace to local",
	Long:  `Download workflow definitions from a workspace to local JSON files. Saves one JSON file per workflow (named by slug) to the output directory. Pass a slug as a positional argument or via --slug to pull a single workflow, or omit to pull all.`,
	Example: `  # Pull all workflows to default directory (suprsend/workflows/)
  suprsend workflow pull

  # Pull a single workflow by slug
  suprsend workflow pull welcome

  # Pull to a custom directory using the flag form
  suprsend workflow pull --slug welcome --dir ./my-workflows`,
	Annotations: map[string]string{
		"skills:tip.a-overwrite": "Pull overwrites local workflow JSON files for the matched slugs. Commit local edits first if you don't want them clobbered (or use `--force` to skip the prompt).",
		"skills:tip.b-mode":      "Defaults to the **live** mode. Use `--mode draft` to mirror the pending state instead.",
	},
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		mode, _ := cmd.Flags().GetString("mode")
		outputDir, _ := cmd.Flags().GetString("dir")
		slug := utils.ResolveSlug(cmd, args)
		force, _ := cmd.Flags().GetBool("force")
		if outputDir == "" {
			outputDir = filepath.Join(".", "suprsend", "workflows")
			if _, err := os.Stat(outputDir); os.IsNotExist(err) {
				if force {
					log.Infof("Using default directory: %s", outputDir)
				} else {
					od, success := promptForOutputDirectory()
					if !success {
						return nil
					}
					outputDir = od
				}
			}
			if outputDir == "" {
				return clierr.New("no output directory specified", clierr.CodeInvalidUsage)
			}
		}
		if err := ensureOutputDirectory(outputDir); err != nil {
			log.Errorf("Error with output directory: %v", err)
			return clierr.Wrap(err, clierr.CodeFileNotFound, "")
		}
		spinner := utils.NewSpinner("Loading...")

		mgmntClient := utils.GetSuprSendMgmntClient()
		if slug != "" {
			workflowResp, err := mgmntClient.GetWorkflowDetailBySlug(cmd.Context(), workspace, slug, mode)
			if err != nil {
				log.Errorf("Failed to get workflow detail: %v", err)
				return clierr.Wrap(err, clierr.CodeAPIInternal, "")
			}
			workflowJson, err := json.MarshalIndent(workflowResp, "", "  ")
			if err != nil {
				log.Errorf("Failed to marshal workflow: %v", err)
				return clierr.Wrap(err, clierr.CodeFileParseFailed, "")
			}
			slugDir := filepath.Join(outputDir, slug)
			if err := os.MkdirAll(slugDir, 0o755); err != nil {
				log.Errorf("Failed to create workflow directory: %v", err)
				return clierr.Wrap(err, clierr.CodeFileNotFound, "")
			}
			if err := os.WriteFile(filepath.Join(slugDir, "workflow.json"), append(workflowJson, '\n'), 0o644); err != nil {
				log.Errorf("Failed to write workflow file: %v", err)
				return clierr.Wrap(err, clierr.CodeFileParseFailed, "")
			}
			spinner.Stop(fmt.Sprintf("Pulled %s from %s", slug, workspace))
			return nil
		}

		workflows_resp, err := mgmntClient.GetWorkflows(cmd.Context(), workspace, mode)
		if err != nil {
			log.Errorf("Failed to get workflows: %v", err)
			return clierr.Wrap(err, clierr.CodeAPIInternal, "")
		}
		spinner.Stop(fmt.Sprintf("Pulled %d workflows from %s", len(workflows_resp.Results), workspace))

		stats, err := WriteWorkflowsToFiles(*workflows_resp, outputDir)
		if err != nil {
			log.Errorf("Failed to save workflows: %v", err)
			return clierr.Wrap(err, clierr.CodeFileParseFailed, "")
		}

		log.Info("=== Workflow Pull Summary ===")
		log.Infof("Total workflows processed: %d", stats.Total)
		log.Infof("Successfully updated: %d", stats.Success)
		log.Infof("Failed to pull: %d", stats.Failed)

		if stats.Failed > 0 {
			log.Info("Failed workflows:")
			for _, errorMsg := range stats.Errors {
				log.Infof("  - %s", errorMsg)
			}
		}
		return nil
	},
}

func init() {
	workflowPullCmd.PersistentFlags().StringP("mode", "m", "live", "Version mode: draft or live")
	workflowPullCmd.PersistentFlags().StringP("dir", "d", "", "Directory to save workflow files to (default: ./suprsend/workflows)")
	workflowPullCmd.PersistentFlags().StringP("slug", "g", "", "Workflow slug to pull (omit to pull all)")
	workflowPullCmd.PersistentFlags().BoolP("force", "F", false, "Skip directory confirmation prompt, use default path")
	WorkflowCmd.AddCommand(workflowPullCmd)
}
