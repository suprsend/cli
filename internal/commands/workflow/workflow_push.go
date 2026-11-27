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
		"skills:tip.a-draft":   "Push writes to the **draft** state. Run `suprsend workflow commit` to promote draft → live, or pass `--commit` to push + commit in one step.",
		"skills:tip.b-dryrun":  "`--dry-run` makes no API calls: it only reads and parses the local files (or `--json`) and reports what would be pushed. Nothing is validated server-side.",
		"skills:tip.c-commit":  "With `--commit`, a workflow that is saved as a draft but not committed is an error. Pushing one workflow (`workflow push <slug> --commit`) exits 1 for an invalid workflow (code `api-validation-failed`, with the validation errors in the message) and for any other commit failure. Pushing all workflows exits 1 if any of them fails, and the error lists each failed workflow with its reason. A workflow whose draft already matches live counts as success (nothing to commit).",
		"skills:tip.d-force":   "`-F/--force` only skips the confirmation prompt for `--commit`; it does not change what is committed. The prompt only appears when stdin is a TTY, so in non-interactive shells (CI, agents) `--commit` proceeds without asking even without `-F`.",
		"skills:tip.e-result":  "Each workflow is reported on stdout as `created` or `updated` (saved as draft), `unchanged` (the workspace's draft already matches the file, so nothing was sent; with `--commit`, the draft already matched live), `skipped` (see the state tip), `committed` or `failed`, followed by a summary; with `-o json` the result is one JSON object with `items` and `counts`.",
		"skills:tip.f-state":   "Pull and push keep `.suprsend-state.json` in the workflows directory (commit it with the files): what each workflow looked like, per workspace, at the last pull or push. Push compares the file and the workspace's draft with it: a workflow changed only in the workspace since then is `skipped` rather than overwritten with the stale file (with `--commit` that is an error, so nobody else's draft goes live unseen), and one changed on both sides fails as a conflict (code `api-conflict`). Pull again to get the workspace's version, or pass `--overwrite` to push the file anyway. Without a record for that workspace (a first push, or files pulled from another workspace), push sends every workflow that differs from the draft. A pull takes the live version by default, while push compares with the draft. If the draft differed from live at the pull, it is someone's pending work: push reports the workflow `unchanged` with a note and leaves the draft alone (with `--commit` it is not committed), and a local edit to it fails as a conflict. Pull with `--mode draft` to work on the draft, or pass `--overwrite` to replace it. The order of a multi-channel node's `channels` is ignored when comparing.",
		"skills:tip.g-enabled": "Enabled/disabled is a property of the workflow, not a version: an `is_enabled` in the file applies on push, without `--commit`. `workflow pull` writes the key. If a file has no `is_enabled`, a push leaves the workflow's enabled state as it is; a new workflow starts enabled. Use `workflow enable` / `workflow disable` to change only that.",
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
		overwrite, _ := cmd.Flags().GetBool("overwrite")
		opts := pushOptions{commit: commit, commitMessage: commitMessage, overwrite: overwrite}

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
				st, err := utils.LoadState(path)
				if err != nil {
					return err
				}
				opts.state = st
			}

			report := utils.NewReport("workflow push", workspace, dryRun)
			if dryRun {
				report.Add("workflows/"+slug, dryRunStatus, "")
				return report.Finish(nil)
			}

			spinner := utils.NewSpinner(fmt.Sprintf("Pushing %s...", slug))
			status, detail, err := pushOne(mgmntClient, workspace, slug, workflow, opts)
			spinner.Stop("")
			saveState(opts.state)
			if err != nil {
				err = clierr.Wrap(err, clierr.CodeAPIInternal, fmt.Sprintf("failed to push workflow %s", slug))
				report.Fail("workflows/"+slug, err)
				return report.Finish(err)
			}
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

		files, err := utils.ReadDir(path)
		if err != nil {
			return clierr.Wrap(err, clierr.CodeFileNotFound, "")
		}

		if !dryRun {
			st, err := utils.LoadState(path)
			if err != nil {
				return err
			}
			opts.state = st
		}

		report := utils.NewReport("workflow push", workspace, dryRun)
		var failures []string
		var failedErrs []error
		fail := func(slug, action string, err error) {
			report.Fail("workflows/"+slug, err)
			failures = append(failures, clierr.ItemFailure("workflows/"+slug, action, err))
			failedErrs = append(failedErrs, err)
		}

		for _, file := range files {
			if !file.IsDir() {
				continue
			}

			slug := file.Name()
			filePath := filepath.Join(path, slug, "workflow.json")
			data, err := os.ReadFile(filePath)
			if err != nil {
				fail(slug, "failed to read workflow.json", clierr.Wrap(err, clierr.CodeFileNotFound, ""))
				continue
			}

			var workflow map[string]any
			if err := json.Unmarshal(data, &workflow); err != nil {
				fail(slug, "failed to parse workflow.json", clierr.Wrap(err, clierr.CodeFileParseFailed, ""))
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
			status, detail, err := pushOne(mgmntClient, workspace, slug, workflow, opts)
			spinner.Stop("")
			if err != nil {
				fail(slug, "failed to push", err)
				continue
			}
			report.Add("workflows/"+slug, status, detail)
		}
		saveState(opts.state)

		if len(failures) > 0 {
			code := clierr.FailuresCode(failedErrs)
			summary := fmt.Sprintf("%d workflow(s) failed to push", len(failures))
			if code == clierr.CodeFileParseFailed {
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
	workflowPushCmd.PersistentFlags().Bool("overwrite", false, "Push your copy even when the workspace's version changed since the last pull or push, replacing that change")
	WorkflowCmd.AddCommand(workflowPushCmd)
}
