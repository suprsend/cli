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

var schemaPullCmd = &cobra.Command{
	Use:   "pull [<slug>]",
	Short: "Pull schemas",
	Long:  `Download schema definitions from a workspace to local JSON files. Saves one JSON file per schema (named by slug) to the output directory. Pass a slug as a positional argument or via --slug to pull a single schema, or omit to pull all.`,
	Args:  cobra.MaximumNArgs(1),
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
						return clierr.New("no output directory specified", clierr.CodeUnknown)
					}
					outputDir = od
				}
			}
			if outputDir == "" {
				return clierr.New("no output directory specified", clierr.CodeUnknown)
			}
		}
		if err := ensureOutputDirectory(outputDir); err != nil {
			log.Errorf("Error with output directory: %v", err)
			return clierr.Wrap(err, clierr.CodeFileNotFound, "")
		}

		workspace, _ := cmd.Flags().GetString("workspace")
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			log.Errorf("Failed to create directory: %v", err)
			return clierr.Wrap(err, clierr.CodeFileNotFound, "")
		}
		spinner := utils.NewSpinner("Loading...")

		mgmntClient := utils.GetSuprSendMgmntClient()
		if slug != "" {
			schema, err := mgmntClient.GetSchemaBySlug(workspace, slug, mode)
			if err != nil {
				log.Errorf("Failed to get schema: %v", err)
				return clierr.Wrap(err, clierr.CodeAPIInternal, "")
			}
			spinner.Stop(fmt.Sprintf("Pulled %s from %s", slug, workspace))
			obj := *schema
			slugDir := filepath.Join(outputDir, slug)
			if err := os.MkdirAll(slugDir, 0o755); err != nil {
				return clierr.Wrap(err, clierr.CodeFileNotFound, "")
			}
			if err := writeSchemaFiles(slugDir, slug, obj); err != nil {
				return clierr.Wrap(err, clierr.CodeFileParseFailed, "")
			}
			log.Infof("Wrote schema to %s", filepath.Join(slugDir, "schema.json"))
			return nil
		}
		schemas, err := mgmntClient.GetSchemas(workspace, mode)
		if err != nil {
			log.Errorf("Failed to get schemas: %v", err)
			return clierr.Wrap(err, clierr.CodeAPIInternal, "")
		}
		spinner.Stop(fmt.Sprintf("Pulled %d schemas from %s", len(schemas.Results), workspace))
		stats, err := WriteSchemasToFiles(schemas, outputDir)
		if err != nil {
			log.Errorf("Failed to save schemas: %v", err)
			return clierr.Wrap(err, clierr.CodeFileParseFailed, "")
		}

		log.Info("=== Schema Pull Summary ===")
		log.Infof("Total schemas processed: %d", stats.Total)
		log.Infof("Successfully updated: %d", stats.Success)
		log.Infof("Failed to pull: %d", stats.Failed)

		if stats.Failed > 0 {
			log.Info("Failed schemas:")
			for _, errorMsg := range stats.Errors {
				log.Infof("  - %s", errorMsg)
			}
		}
		return nil
	},
}

func init() {
	schemaPullCmd.Flags().StringP("dir", "d", "", "Directory to save schema files to (default: ./suprsend/schemas)")
	schemaPullCmd.PersistentFlags().StringP("mode", "m", "live", "Version mode: draft or live")
	schemaPullCmd.PersistentFlags().StringP("slug", "g", "", "Schema slug to pull (omit to pull all)")
	schemaPullCmd.PersistentFlags().BoolP("force", "F", false, "Skip directory confirmation prompt, use default path")
	SchemaCmd.AddCommand(schemaPullCmd)
}
