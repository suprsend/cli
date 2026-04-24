package schema

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/utils"
	"github.com/yarlson/pin"
)

var schemaPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull schemas",
	Long:  `Download schema definitions from a workspace to local JSON files. Saves one JSON file per schema (named by slug) to the output directory. Use --slug to pull a single schema, or omit to pull all.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		outputDir, _ := cmd.Flags().GetString("dir")
		mode, _ := cmd.Flags().GetString("mode")
		slug, _ := cmd.Flags().GetString("slug")
		force, _ := cmd.Flags().GetBool("force")

		if outputDir == "" {
			outputDir = filepath.Join(".", "suprsend", "schemas")
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

		workspace, _ := cmd.Flags().GetString("workspace")
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			fmt.Fprintf(os.Stdout, "Error: Failed to create directory: %v\n", err)
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
			schema, err := mgmntClient.GetSchemaBySlug(workspace, slug, mode)
			if err != nil {
				fmt.Fprintf(os.Stdout, "Error: Failed to get schema: %v\n", err)
				return err
			}
			if p != nil {
				p.Stop(fmt.Sprintf("Pulled %s from %s", slug, workspace))
			}
			obj := *schema
			slugDir := filepath.Join(outputDir, slug)
			if err := os.MkdirAll(slugDir, 0o755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", slugDir, err)
			}
			if err := writeSchemaFiles(slugDir, slug, obj); err != nil {
				return fmt.Errorf("failed to write schema files: %w", err)
			}
			fmt.Fprintf(os.Stdout, "Wrote schema to %s\n", filepath.Join(slugDir, "schema.json"))
			return nil
		}
		schemas, err := mgmntClient.GetSchemas(workspace, mode)
		if err != nil {
			fmt.Fprintf(os.Stdout, "Error: Failed to get schemas: %v\n", err)
			return err
		}
		if p != nil {
			p.Stop(fmt.Sprintf("Pulled %d schemas from %s", len(schemas.Results), workspace))
		}
		stats, err := WriteSchemasToFiles(schemas, outputDir)
		if err != nil {
			fmt.Fprintf(os.Stdout, "Error: Failed to save schemas: %v\n", err)
			return err
		}

		fmt.Fprintf(os.Stdout, "\n=== Schema Pull Summary ===\n")
		fmt.Fprintf(os.Stdout, "Total schemas processed: %d\n", stats.Total)
		fmt.Fprintf(os.Stdout, "Successfully updated: %d\n", stats.Success)
		fmt.Fprintf(os.Stdout, "Failed to pull: %d\n", stats.Failed)

		if stats.Failed > 0 {
			fmt.Fprintf(os.Stdout, "\nFailed schemas:\n")
			for _, errorMsg := range stats.Errors {
				fmt.Fprintf(os.Stdout, "  - %s\n", errorMsg)
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
