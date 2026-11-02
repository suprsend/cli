package schema

import (
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/utils"
)

var schemaPushCmd = &cobra.Command{
	Use:   "push [<slug>]",
	Short: "Push schemas",
	Long:  "Upload local schemas to a workspace. For every subdirectory `<dir>/<slug>/` of the input directory (default `./suprsend/schemas`), reads `schema.json` (name, description, and `is_enabled` if present) and `payload_schema.json` (the JSON Schema), validates the payload schema locally, and pushes both; the directory name is used as the slug. By default, changes are staged as drafts. Use --commit to also promote to live. Pass a slug as a positional argument or via --slug to push a single schema, or omit to push all. With a slug, --json takes the JSON Schema inline instead of reading the slug's files.",
	Example: `  # Push all schemas from default directory
  suprsend schema push

  # Push a single schema and commit to live immediately
  suprsend schema push order-placed --commit

  # Push an inline JSON Schema instead of reading payload_schema.json
  suprsend schema push order-placed --json '{"type":"object","properties":{"order_id":{"type":"string"}},"required":["order_id"]}'

  # Dry run: preview what would be pushed without making changes
  suprsend schema push --dry-run`,
	Annotations: map[string]string{
		"skills:tip.a-draft":   "Push writes to the **draft** state. Run `suprsend schema commit` to promote draft → live, or pass `--commit` to push + commit in one step.",
		"skills:tip.b-dryrun":  "`--dry-run` makes no API calls. It reads the local files, compiles each payload schema (`payload_schema.json`, or the `--json` value) locally as JSON Schema (draft 2020-12; root `type` must be `object`), and reports what would be pushed. Nothing is validated server-side.",
		"skills:tip.c-regen":   "After committing a schema change, regenerate types with `suprsend generate-types <language>` so consuming code stays in sync.",
		"skills:tip.d-commit":  "With `--commit`, a schema that is saved as a draft but not committed (for example an empty JSON Schema) is an error: the command exits non-zero with the API's reason. A schema whose draft already matches live counts as success (nothing to commit).",
		"skills:tip.f-enabled": "Enabled/disabled is a property of the schema, not a version: it applies on push, without `--commit`. If `schema.json` has no `is_enabled` key, the API currently **enables** the schema on every push, re-enabling a disabled one (this includes `sync`). To keep a schema disabled, set `\"is_enabled\": false` in its `schema.json`. `--json` pushes never send `is_enabled`.",
		"skills:tip.g-result":  "Each schema is reported on stdout as `pushed` (draft only), `committed`, `unchanged` (the draft already matched live) or `failed`, followed by a summary; with `-o json` the result is one JSON object with `items` and `counts`.",
		"skills:tip.e-force":   "`-F/--force` only skips the confirmation prompt for `--commit`; it does not change what is committed. The prompt only appears when stdin is a TTY, so in non-interactive shells (CI, agents) `--commit` proceeds without asking even without `-F`.",
	},
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		slug := utils.ResolveSlug(cmd, args)
		commit, _ := cmd.Flags().GetBool("commit")
		commitMessage, _ := cmd.Flags().GetString("commit-message")
		path, _ := cmd.Flags().GetString("dir")
		jsonPayload, _ := cmd.Flags().GetString("json")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		force, _ := cmd.Flags().GetBool("force")

		if jsonPayload != "" && slug == "" {
			return clierr.New("--json requires --slug to be specified", clierr.CodeInvalidUsage)
		}

		if commit && !dryRun && !force {
			target := "all schemas"
			if slug != "" {
				target = fmt.Sprintf("schema '%s'", slug)
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
			var schema map[string]any
			if jsonPayload != "" {
				payload, err := SchemaPayloadFromJSON(slug, jsonPayload)
				if err != nil {
					return clierr.Wrap(err, clierr.CodeFileParseFailed, "")
				}
				schema = payload
			} else {
				if path == "" {
					path = filepath.Join(".", "suprsend", "schemas")
				}
				if _, err := os.Stat(path); os.IsNotExist(err) {
					return clierr.Wrap(err, clierr.CodeFileNotFound, fmt.Sprintf("directory %s does not exist", path))
				}
				if err := validateInputDirectory(path); err != nil {
					return clierr.Wrap(err, clierr.CodeFileNotFound, "error with input directory")
				}

				merged, err := ReadAndMergeSchemaFiles(filepath.Join(path, slug), slug)
				if err != nil {
					return clierr.Wrap(err, clierr.CodeFileNotFound, fmt.Sprintf("failed to read schema files for %s", slug))
				}
				schema = merged
			}

			report := utils.NewReport("schema push", workspace, dryRun)
			if dryRun {
				report.Add("schemas/"+slug, dryRunStatus, "")
				return report.Finish(nil)
			}

			spinner := utils.NewSpinner(fmt.Sprintf("Pushing %s...", slug))
			commitStatus, err := mgmntClient.PushSchema(workspace, slug, schema, commit, commitMessage)
			spinner.Stop("")
			if err != nil {
				err = clierr.Wrap(err, clierr.CodeAPIInternal, fmt.Sprintf("failed to push schema %s", slug))
				report.Fail("schemas/"+slug, err)
				return report.Finish(err)
			}
			status, detail := utils.PushStatus(commitStatus)
			report.Add("schemas/"+slug, status, detail)
			return report.Finish(nil)
		}

		if path == "" {
			path = filepath.Join(".", "suprsend", "schemas")
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			return clierr.Wrap(err, clierr.CodeFileNotFound, fmt.Sprintf("directory %s does not exist", path))
		}

		if err := validateInputDirectory(path); err != nil {
			return clierr.Wrap(err, clierr.CodeFileNotFound, "error with input directory")
		}

		entries, err := os.ReadDir(path)
		if err != nil {
			return clierr.Wrap(err, clierr.CodeFileNotFound, "")
		}

		report := utils.NewReport("schema push", workspace, dryRun)
		var failures []string
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			slug := entry.Name()

			schema, err := ReadAndMergeSchemaFiles(filepath.Join(path, slug), slug)
			if err != nil {
				report.Fail("schemas/"+slug, err)
				failures = append(failures, err.Error())
				continue
			}

			if dryRun {
				report.Add("schemas/"+slug, dryRunStatus, "")
				continue
			}

			spinner := utils.NewSpinner(fmt.Sprintf("Pushing %s...", slug))
			commitStatus, err := mgmntClient.PushSchema(workspace, slug, schema, commit, commitMessage)
			spinner.Stop("")
			if err != nil {
				report.Fail("schemas/"+slug, err)
				failures = append(failures, clierr.ItemFailure("schemas/"+slug, "failed to push", err))
				continue
			}
			status, detail := utils.PushStatus(commitStatus)
			report.Add("schemas/"+slug, status, detail)
		}

		if len(failures) > 0 {
			code := clierr.CodeAPIInternal
			summary := fmt.Sprintf("%d schema(s) failed to push", len(failures))
			if dryRun {
				code = clierr.CodeFileParseFailed
				summary = fmt.Sprintf("%d schema(s) failed local checks", len(failures))
			}
			return report.Finish(clierr.Failures(summary, failures, code))
		}
		return report.Finish(nil)
	},
}

func init() {
	schemaPushCmd.Flags().StringP("dir", "d", "", "Directory containing schema files (default: ./suprsend/schemas)")
	schemaPushCmd.Flags().BoolP("commit", "c", false, "Promote changes from draft to live after pushing")
	schemaPushCmd.Flags().String("commit-message", "", "Message describing the changes being committed")
	schemaPushCmd.PersistentFlags().StringP("slug", "g", "", "Schema slug to push (omit to push all)")
	schemaPushCmd.PersistentFlags().StringP("json", "j", "", `Payload JSON Schema as a JSON object, used instead of payload_schema.json (requires --slug). Must be a valid JSON Schema with root type "object", e.g. '{"type":"object","properties":{"key":{"type":"string"}}}'`)
	schemaPushCmd.Flags().BoolP("dry-run", "n", false, "Report what would be pushed without making any changes")
	schemaPushCmd.Flags().BoolP("force", "F", false, "Skip confirmation prompt when --commit is set")
	SchemaCmd.AddCommand(schemaPushCmd)
}
