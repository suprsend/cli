package schema

import (
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

var schemaPullCmd = &cobra.Command{
	Use:   "pull [<slug>]",
	Short: "Pull schemas",
	Long:  "Download schema definitions from a workspace to local JSON files. Each schema is saved as `<dir>/<slug>/schema.json` (slug, name, description, and `is_enabled` when the API returns it) plus `<dir>/<slug>/payload_schema.json` (the JSON Schema), default dir `./suprsend/schemas`. Pass a slug as a positional argument or via --slug to pull a single schema, or omit to pull all.",
	Example: `  # Pull all schemas to default directory (suprsend/schemas/)
  suprsend schema pull

  # Pull a single schema by slug
  suprsend schema pull order-placed

  # Pull to a custom directory using the flag form
  suprsend schema pull --slug order-placed --dir ./my-schemas`,
	Annotations: map[string]string{
		"skills:tip.a-overwrite": "Pull overwrites `<dir>/<slug>/schema.json` and `payload_schema.json` for every pulled slug without asking, so commit or stash local edits first. `--force` does not affect overwriting: it only skips the output-directory prompt shown when `--dir` is omitted and `./suprsend/schemas` doesn't exist. In non-interactive shells pass `--dir` or `--force`; otherwise the command exits 64 (invalid usage) without pulling anything.",
		"skills:tip.b-mode":      "Defaults to the **live** mode. Use `--mode draft` to mirror the pending state instead.",
	},
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		outputDir, _ := cmd.Flags().GetString("dir")
		mode, _ := cmd.Flags().GetString("mode")
		slug := utils.ResolveSlug(cmd, args)
		force, _ := cmd.Flags().GetBool("force")

		if outputDir == "" {
			outputDir = filepath.Join(".", "suprsend", "schemas")
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
				return clierr.New("no output directory specified", clierr.CodeUnknown)
			}
		}
		if err := ensureOutputDirectory(outputDir); err != nil {
			return clierr.Wrap(err, clierr.CodeFileNotFound, "")
		}

		workspace, _ := cmd.Flags().GetString("workspace")
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return clierr.Wrap(err, clierr.CodeFileNotFound, "")
		}
		report := utils.NewReport("schema pull", workspace, false)
		spinner := utils.NewSpinner("Loading...")

		mgmntClient := utils.GetSuprSendMgmntClient()
		if slug != "" {
			err := pullOneSchema(mgmntClient, workspace, mode, slug, outputDir)
			spinner.Stop("")
			if err != nil {
				report.Fail("schemas/"+slug, err)
				return report.Finish(err)
			}
			report.Add("schemas/"+slug, utils.StatusPulled, "")
			return report.Finish(nil)
		}
		schemas, err := mgmntClient.GetSchemas(workspace, mode)
		spinner.Stop("")
		if err != nil {
			return report.Finish(clierr.Wrap(err, clierr.CodeAPIInternal, ""))
		}
		stats, err := WriteSchemasToFiles(schemas, outputDir)
		if err != nil {
			return report.Finish(clierr.Wrap(err, clierr.CodeFileParseFailed, ""))
		}
		for _, s := range stats.Written {
			report.Add("schemas/"+s, utils.StatusPulled, "")
		}
		for _, e := range stats.Errors {
			report.Fail("schemas", errors.New(e))
		}
		if stats.Failed > 0 {
			return report.Finish(clierr.Failures(fmt.Sprintf("%d schema(s) failed to pull", stats.Failed), stats.Errors, clierr.CodeAPIInternal))
		}
		return report.Finish(nil)
	},
}

// pullOneSchema writes the files of one schema under <outputDir>/<slug>/.
func pullOneSchema(mgmntClient *mgmnt.SS_MgmntClient, workspace, mode, slug, outputDir string) error {
	schema, err := mgmntClient.GetSchemaBySlug(workspace, slug, mode)
	if err != nil {
		return clierr.Wrap(err, clierr.CodeAPIInternal, "")
	}
	slugDir := filepath.Join(outputDir, slug)
	if err := os.MkdirAll(slugDir, 0o755); err != nil {
		return clierr.Wrap(err, clierr.CodeFileNotFound, "")
	}
	if err := writeSchemaFiles(slugDir, slug, *schema); err != nil {
		return clierr.Wrap(err, clierr.CodeFileParseFailed, "")
	}
	return nil
}

func init() {
	schemaPullCmd.Flags().StringP("dir", "d", "", "Directory to save schema files to (default: ./suprsend/schemas)")
	schemaPullCmd.PersistentFlags().StringP("mode", "m", "live", "Version mode: draft or live")
	schemaPullCmd.PersistentFlags().StringP("slug", "g", "", "Schema slug to pull (omit to pull all)")
	schemaPullCmd.PersistentFlags().BoolP("force", "F", false, "Skip directory confirmation prompt, use default path")
	SchemaCmd.AddCommand(schemaPullCmd)
}
