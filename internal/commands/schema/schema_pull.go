package schema

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

var schemaPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull schemas",
	Long:  `Pull schemas in a workspace`,
	RunE: func(cmd *cobra.Command, args []string) error {
		outputDir, _ := cmd.Flags().GetString("dir")
		mode, _ := cmd.Flags().GetString("mode")
		slug, _ := cmd.Flags().GetString("slug")
		force, _ := cmd.Flags().GetBool("force")

		if outputDir == "" {
			outputDir = filepath.Join(".", "suprsend", "schema")
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
			schemaData, err := json.MarshalIndent(schema, "", "  ")
			if err != nil {
				fmt.Fprintf(os.Stdout, "Error: Failed to marshal schema: %v\n", err)
				return err
			}
			if p != nil {
				p.Stop(fmt.Sprintf("Pulled %s from %s", slug, workspace))
			}
			if err := os.WriteFile(filepath.Join(outputDir, slug+".json"), schemaData, 0644); err != nil {
				return fmt.Errorf("failed to write schema file: %w", err)
			}
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
	schemaPullCmd.Flags().StringP("dir", "d", "", "Directory to pull schemas (default: ./suprsend/schema)")
	schemaPullCmd.PersistentFlags().StringP("mode", "m", "live", "Mode of schemas to pull (draft, live), default: live")
	schemaPullCmd.PersistentFlags().StringP("slug", "g", "", "Slug of schema to pull")
	schemaPullCmd.PersistentFlags().BoolP("force", "f", false, "Force using default directory without prompting")
	SchemaCmd.AddCommand(schemaPullCmd)
}
