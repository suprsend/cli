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
		"skills:tip.b-dryrun": "`--dry-run` makes no API calls: it only reads and parses the local files (or `--json`) and logs what would be pushed. Nothing is validated server-side.",
		"skills:tip.c-commit": "With `--commit`, a workflow that is saved as a draft but not committed is an error. Pushing one workflow (`workflow push <slug> --commit`) exits 2 for an invalid workflow, with the validation errors in the message, and 1 for any other commit failure. Pushing all workflows exits 1 if any of them fails; the per-workflow reasons are only in the log output. A workflow whose draft already matches live counts as success (nothing to commit).",
		"skills:tip.d-force":  "`-F/--force` only skips the confirmation prompt for `--commit`; it does not change what is committed. The prompt only appears when stdin is a TTY, so in non-interactive shells (CI, agents) `--commit` proceeds without asking even without `-F`.",
	},
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		path, _ := cmd.Flags().GetString("dir")
		commit, _ := cmd.Flags().GetBool("commit")
		commitMessage, _ := cmd.Flags().GetString("commit-message")
		slug := utils.ResolveSlug(cmd, args)
		jsonPayload, _ := cmd.Flags().GetString("json")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		force, _ := cmd.Flags().GetBool("force")
		var dryRunSlugs []string

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

		hasError := false
		var spinner *utils.Spinner

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
				// path wins: --slug flag value is authoritative
				workflow["slug"] = slug
			}

			if dryRun {
				action := "push"
				if commit {
					action = "push and commit"
				}
				log.Infof("DRY RUN: would %s workflow '%s' to %s", action, slug, workspace)
				return nil
			}

			spinner = utils.NewSpinner(fmt.Sprintf("Pushing %s...", slug))
			err := mgmntClient.PushWorkflow(workspace, slug, workflow, commit, commitMessage)
			if err != nil {
				spinner.Stop("")
				return clierr.Wrap(err, clierr.CodeAPIInternal, fmt.Sprintf("failed to push workflow %s", slug))
			}
			spinner.Stop(fmt.Sprintf("Pushed workflow: %s", slug))
			return nil
		}

		stats := &WorkflowPushStats{
			Errors: []string{},
		}

		if path == "" {
			path = filepath.Join(".", "suprsend", "workflows")
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			log.Errorf("Directory %s does not exist", path)
			return clierr.Wrap(err, clierr.CodeFileNotFound, "")
		}

		if err := validateInputDirectory(path); err != nil {
			log.Errorf("Error with input directory: %v\n", err)
			return clierr.Wrap(err, clierr.CodeFileNotFound, "")
		}

		files, err := os.ReadDir(path)
		if err != nil {
			log.WithError(err).Errorf("Failed to read local workflows directory")
			return clierr.Wrap(err, clierr.CodeFileNotFound, "")
		}

		for _, file := range files {
			if file.IsDir() {
				stats.Total++
			}
		}

		for _, file := range files {
			if !file.IsDir() {
				continue
			}

			slug := file.Name()
			if !hasError {
				spinner = utils.NewSpinner(fmt.Sprintf("Pushing %s...", slug))
			}
			filePath := filepath.Join(path, slug, "workflow.json")
			data, err := os.ReadFile(filePath)
			if err != nil {
				spinner.Stop("")
				hasError = true
				log.WithError(err).Errorf("Failed to read file %s", file.Name())
				stats.Failed++
				stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to read file %s: %v", file.Name(), err))
				continue
			}

			var workflow map[string]any
			if err := json.Unmarshal(data, &workflow); err != nil {
				spinner.Stop("")
				hasError = true
				log.WithError(err).Errorf("Failed to parse JSON for %s", file.Name())
				stats.Failed++
				stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to parse JSON for %s: %v", file.Name(), err))
				continue
			}

			// path wins: directory name is authoritative; warn on mismatch
			if jsonSlug, _ := workflow["slug"].(string); jsonSlug != slug {
				log.Warnf("workflows/%s: workflow.json#slug is %q but directory is %q — using directory value", slug, jsonSlug, slug)
				workflow["slug"] = slug
			}

			if dryRun {
				dryRunSlugs = append(dryRunSlugs, slug)
				stats.Success++
				spinner.Stop(fmt.Sprintf("(dry run) %s", slug))
				hasError = false
				continue
			}

			err = mgmntClient.PushWorkflow(workspace, slug, workflow, commit, commitMessage)
			if err != nil {
				spinner.Stop("")
				hasError = true
				log.WithError(err).Errorf("workflows/%s: failed to push", slug)
				stats.Failed++
				stats.Errors = append(stats.Errors, fmt.Sprintf("workflows/%s: failed to push: %v", slug, err))
				continue
			}

			stats.Success++
			spinner.Stop(fmt.Sprintf("Pushed workflow: %s", slug))
			hasError = false
		}

		if dryRun {
			action := "push"
			if commit {
				action = "push and commit"
			}
			log.Infof("DRY RUN: would %s %d workflow(s) to %s", action, len(dryRunSlugs), workspace)
			for _, s := range dryRunSlugs {
				log.Infof("  - %s", s)
			}
			if stats.Failed > 0 {
				return clierr.New(fmt.Sprintf("%d workflow(s) failed local checks", stats.Failed), clierr.CodeFileParseFailed)
			}
			return nil
		}

		log.Info("=== Workflow Push Summary ===")
		log.Infof("Total workflows processed: %d", stats.Total)
		log.Infof("Successfully pushed: %d", stats.Success)
		log.Infof("Failed to push: %d", stats.Failed)

		if stats.Failed > 0 {
			log.Info("Failed workflows:")
			for _, errorMsg := range stats.Errors {
				log.Infof("  - %s", errorMsg)
			}
			return clierr.New(fmt.Sprintf("%d workflow(s) failed to push", stats.Failed), clierr.CodeAPIInternal)
		}
		return nil
	},
}

func init() {
	workflowPushCmd.PersistentFlags().StringP("dir", "d", "", "Directory containing workflow subdirectories (default: ./suprsend/workflows)")
	workflowPushCmd.PersistentFlags().BoolP("commit", "c", false, "Promote changes from draft to live after pushing")
	workflowPushCmd.PersistentFlags().String("commit-message", "", "Message describing the changes being committed")
	workflowPushCmd.PersistentFlags().StringP("slug", "g", "", "Workflow slug to push (omit to push all)")
	workflowPushCmd.PersistentFlags().StringP("json", "j", "", `Workflow definition as a JSON object (requires --slug). Must be a valid workflow object, e.g. '{"name":"My Workflow","nodes":[...]}'`)
	workflowPushCmd.PersistentFlags().BoolP("dry-run", "n", false, "Print what would be pushed without making any changes")
	workflowPushCmd.PersistentFlags().BoolP("force", "F", false, "Skip confirmation prompt when --commit is set")
	WorkflowCmd.AddCommand(workflowPushCmd)
}
