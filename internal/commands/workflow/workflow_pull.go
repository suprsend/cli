package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/cli/mgmnt"
)

var workflowPullCmd = &cobra.Command{
	Use:   "pull [<slug>]",
	Short: "Pull workflows from SuprSend workspace to local",
	Long:  "Download workflow definitions from a workspace to local JSON files. Each workflow is saved as `<dir>/<slug>/workflow.json` (default dir `./suprsend/workflows`). Pass a slug as a positional argument or via --slug to pull a single workflow, or omit to pull all.",
	Example: `  # Pull all workflows to default directory (suprsend/workflows/)
  suprsend workflow pull

  # Pull a single workflow by slug
  suprsend workflow pull welcome

  # Pull to a custom directory using the flag form
  suprsend workflow pull --slug welcome --dir ./my-workflows`,
	Annotations: map[string]string{
		"skills:tip.a-overwrite": "Pull overwrites `<dir>/<slug>/workflow.json` for every pulled slug without asking, so commit or stash local edits first. `--force` does not affect overwriting: it only skips the output-directory prompt shown when `--dir` is omitted and `./suprsend/workflows` doesn't exist. In non-interactive shells pass `--dir` or `--force`; otherwise the command exits 64 (invalid usage) without pulling anything.",
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
						return clierr.New("output directory is required; pass --dir, or --force to use the default", clierr.CodeInvalidUsage)
					}
					outputDir = od
				}
			}
			if outputDir == "" {
				return clierr.New("no output directory specified", clierr.CodeInvalidUsage)
			}
		}
		if err := ensureOutputDirectory(outputDir); err != nil {
			return clierr.Wrap(err, clierr.CodeFileNotFound, "")
		}
		report := utils.NewReport("workflow pull", workspace, false)
		spinner := utils.NewSpinner("Loading...")

		mgmntClient := utils.GetSuprSendMgmntClient()
		if slug != "" {
			err := pullOneWorkflow(mgmntClient, workspace, mode, slug, outputDir)
			spinner.Stop("")
			if err != nil {
				report.Fail("workflows/"+slug, err)
				return report.Finish(err)
			}
			report.Add("workflows/"+slug, utils.StatusPulled, "")
			return report.Finish(nil)
		}

		workflows_resp, err := mgmntClient.GetWorkflows(workspace, mode)
		spinner.Stop("")
		if err != nil {
			return report.Finish(clierr.Wrap(err, clierr.CodeAPIInternal, ""))
		}

		stats, err := WriteWorkflowsToFiles(*workflows_resp, outputDir)
		if err != nil {
			return report.Finish(clierr.Wrap(err, clierr.CodeFileParseFailed, ""))
		}
		for _, s := range stats.Written {
			report.Add("workflows/"+s, utils.StatusPulled, "")
		}
		for _, e := range stats.Errors {
			report.Fail("workflows", errors.New(e))
		}
		if stats.Failed > 0 {
			return report.Finish(clierr.Failures(fmt.Sprintf("%d workflow(s) failed to pull", stats.Failed), stats.Errors, clierr.CodeAPIInternal))
		}
		return report.Finish(nil)
	},
}

// pullOneWorkflow writes <outputDir>/<slug>/workflow.json.
func pullOneWorkflow(mgmntClient *mgmnt.SS_MgmntClient, workspace, mode, slug, outputDir string) error {
	workflowResp, err := mgmntClient.GetWorkflowDetailBySlug(workspace, slug, mode)
	if err != nil {
		return clierr.Wrap(err, clierr.CodeAPIInternal, "")
	}
	workflowJson, err := json.MarshalIndent(workflowResp, "", "  ")
	if err != nil {
		return clierr.Wrap(err, clierr.CodeFileParseFailed, "")
	}
	slugDir := filepath.Join(outputDir, slug)
	if err := os.MkdirAll(slugDir, 0o755); err != nil {
		return clierr.Wrap(err, clierr.CodeFileNotFound, "")
	}
	if err := os.WriteFile(filepath.Join(slugDir, "workflow.json"), append(workflowJson, '\n'), 0o644); err != nil {
		return clierr.Wrap(err, clierr.CodeFileParseFailed, "")
	}
	return nil
}

func init() {
	workflowPullCmd.PersistentFlags().StringP("mode", "m", "live", "Version mode: draft or live")
	workflowPullCmd.PersistentFlags().StringP("dir", "d", "", "Directory to save workflow files to (default: ./suprsend/workflows)")
	workflowPullCmd.PersistentFlags().StringP("slug", "g", "", "Workflow slug to pull (omit to pull all)")
	workflowPullCmd.PersistentFlags().BoolP("force", "F", false, "Skip directory confirmation prompt, use default path")
	WorkflowCmd.AddCommand(workflowPullCmd)
}
