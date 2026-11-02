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

var workflowPushCmd = &cobra.Command{
	Use:   "push [<slug>]",
	Short: "Push workflows from local to SuprSend workspace",
	Long:  "Upload local workflow definitions to a workspace. Reads `<dir>/<slug>/workflow.json` for every subdirectory of the input directory (default `./suprsend/workflows`); the directory name is used as the slug. By default, changes are staged as drafts. Use --commit to also promote to live. Pass a slug as a positional argument or via --slug to push a single workflow, or omit to push all. With --slug, --json pushes an inline workflow definition instead of reading a file.",
	Example: `  # Push all workflows from default directory
  suprsend workflow push

  # Push a single workflow and commit to live immediately
  suprsend workflow push welcome --commit

  # Dry run: preview what would be pushed without making changes
  suprsend workflow push --dry-run`,
	Annotations: map[string]string{
		"skills:tip.a-draft":  "Push writes to the **draft** state. Run `suprsend workflow commit` to promote draft → live, or pass `--commit` to push + commit in one step.",
		"skills:tip.b-dryrun": "`--dry-run` makes no API calls: it only reads and parses the local files (or `--json`) and reports what would be pushed. Nothing is validated server-side.",
		"skills:tip.c-commit": "With `--commit`, a workflow that is saved as a draft but not committed is an error. Pushing one workflow (`workflow push <slug> --commit`) exits 2 for an invalid workflow, with the validation errors in the message, and 1 for any other commit failure. Pushing all workflows exits 1 if any of them fails, and the error lists each failed workflow with its reason. A workflow whose draft already matches live counts as success (nothing to commit).",
		"skills:tip.d-force":  "`-F/--force` only skips the confirmation prompt for `--commit`; it does not change what is committed. The prompt only appears when stdin is a TTY, so in non-interactive shells (CI, agents) `--commit` proceeds without asking even without `-F`.",
		"skills:tip.e-result": "Each workflow is reported on stdout as `pushed` (draft only), `committed`, `unchanged` (the draft already matched live) or `failed`, followed by a summary; with `-o json` the result is one JSON object with `items` and `counts`.",
	},
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		path, _ := cmd.Flags().GetString("dir")
		commit, _ := cmd.Flags().GetBool("commit")
		commitMessage, _ := cmd.Flags().GetString("commit-message")
		slug := utils.ResolveSlug(cmd, args)
		jsonPayload, _ := cmd.Flags().GetString("json")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		force, _ := cmd.Flags().GetBool("force")

		if jsonPayload != "" && slug == "" {
			return clierr.New("--json requires --slug to be specified", clierr.CodeInvalidUsage)
		}

		if commit && !dryRun && !force {
			target := "all workflows"
			if slug != "" {
				target = fmt.Sprintf("workflow '%s'", slug)
			}
			msg := fmt.Sprintf("This will push and promote %s to live in workspace \"%s\". Continue?", target, workspace)
			confirmed, err := utils.ConfirmDestructiveAction(msg)
			if err != nil || !confirmed {
				log.Info("Aborted.")
				return nil
			}
		}

		mgmntClient := utils.GetSuprSendMgmntClient()
		dryRunStatus := utils.StatusWouldPush
		if commit {
			dryRunStatus = utils.StatusWouldCommit
		}

		if slug != "" {
			var workflow map[string]any
			if jsonPayload != "" {
				if err := json.Unmarshal([]byte(jsonPayload), &workflow); err != nil {
					return clierr.Wrap(err, clierr.CodeFileParseFailed, "")
				}
			} else {
				if path == "" {
					path = filepath.Join(".", "suprsend", "workflows")
				}
				if _, err := os.Stat(path); os.IsNotExist(err) {
					return clierr.Wrap(err, clierr.CodeFileNotFound, fmt.Sprintf("directory %s does not exist", path))
				}
				if err := validateInputDirectory(path); err != nil {
					return clierr.Wrap(err, clierr.CodeFileNotFound, "error with input directory")
				}

				filePath := filepath.Join(path, slug, "workflow.json")
				data, err := os.ReadFile(filePath)
				if err != nil {
					return clierr.Wrap(err, clierr.CodeFileNotFound, fmt.Sprintf("failed to read workflow file %s", filePath))
				}
				if err := json.Unmarshal(data, &workflow); err != nil {
					return clierr.Wrap(err, clierr.CodeFileParseFailed, fmt.Sprintf("failed to parse JSON for %s", filePath))
				}
				workflow["slug"] = slug
			}

			report := utils.NewReport("workflow push", workspace, dryRun)
			if dryRun {
				report.Add("workflows/"+slug, dryRunStatus, "")
				return report.Finish(nil)
			}

			spinner := utils.NewSpinner(fmt.Sprintf("Pushing %s...", slug))
			commitStatus, err := mgmntClient.PushWorkflow(workspace, slug, workflow, commit, commitMessage)
			spinner.Stop("")
			if err != nil {
				err = clierr.Wrap(err, clierr.CodeAPIInternal, fmt.Sprintf("failed to push workflow %s", slug))
				report.Fail("workflows/"+slug, err)
				return report.Finish(err)
			}
			status, detail := utils.PushStatus(commitStatus)
			report.Add("workflows/"+slug, status, detail)
			return report.Finish(nil)
		}

		if path == "" {
			path = filepath.Join(".", "suprsend", "workflows")
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			return clierr.Wrap(err, clierr.CodeFileNotFound, fmt.Sprintf("directory %s does not exist", path))
		}

		if err := validateInputDirectory(path); err != nil {
			return clierr.Wrap(err, clierr.CodeFileNotFound, "error with input directory")
		}

		files, err := os.ReadDir(path)
		if err != nil {
			return clierr.Wrap(err, clierr.CodeFileNotFound, "")
		}

		report := utils.NewReport("workflow push", workspace, dryRun)
		var failures []string
		fail := func(slug, action string, err error) {
			report.Fail("workflows/"+slug, err)
			failures = append(failures, clierr.ItemFailure("workflows/"+slug, action, err))
		}

		for _, file := range files {
			if !file.IsDir() {
				continue
			}

			slug := file.Name()
			filePath := filepath.Join(path, slug, "workflow.json")
			data, err := os.ReadFile(filePath)
			if err != nil {
				fail(slug, "failed to read workflow.json", err)
				continue
			}

			var workflow map[string]any
			if err := json.Unmarshal(data, &workflow); err != nil {
				fail(slug, "failed to parse workflow.json", err)
				continue
			}

			if jsonSlug, _ := workflow["slug"].(string); jsonSlug != slug {
				log.Warnf("workflows/%s: workflow.json#slug is %q but directory is %q — using directory value", slug, jsonSlug, slug)
				workflow["slug"] = slug
			}

			if dryRun {
				report.Add("workflows/"+slug, dryRunStatus, "")
				continue
			}

			spinner := utils.NewSpinner(fmt.Sprintf("Pushing %s...", slug))
			commitStatus, err := mgmntClient.PushWorkflow(workspace, slug, workflow, commit, commitMessage)
			spinner.Stop("")
			if err != nil {
				fail(slug, "failed to push", err)
				continue
			}
			status, detail := utils.PushStatus(commitStatus)
			report.Add("workflows/"+slug, status, detail)
		}

		if len(failures) > 0 {
			code := clierr.CodeAPIInternal
			summary := fmt.Sprintf("%d workflow(s) failed to push", len(failures))
			if dryRun {
				code = clierr.CodeFileParseFailed
				summary = fmt.Sprintf("%d workflow(s) failed local checks", len(failures))
			}
			return report.Finish(clierr.Failures(summary, failures, code))
		}
		return report.Finish(nil)
	},
}

func init() {
	workflowPushCmd.PersistentFlags().StringP("dir", "d", "", "Directory containing workflow subdirectories (default: ./suprsend/workflows)")
	workflowPushCmd.PersistentFlags().BoolP("commit", "c", false, "Promote changes from draft to live after pushing")
	workflowPushCmd.PersistentFlags().String("commit-message", "", "Message describing the changes being committed")
	workflowPushCmd.PersistentFlags().StringP("slug", "g", "", "Workflow slug to push (omit to push all)")
	workflowPushCmd.PersistentFlags().StringP("json", "j", "", `Workflow definition as a JSON object (requires --slug). Must be a valid workflow object, e.g. '{"name":"My Workflow","nodes":[...]}'`)
	workflowPushCmd.PersistentFlags().BoolP("dry-run", "n", false, "Report what would be pushed without making any changes")
	workflowPushCmd.PersistentFlags().BoolP("force", "F", false, "Skip confirmation prompt when --commit is set")
	WorkflowCmd.AddCommand(workflowPushCmd)
}
