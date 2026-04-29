package schema

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/utils"
)

var schemaGetCmd = &cobra.Command{
	Use:   "get [<slug>]",
	Short: "Get schema details",
	Long:  "Retrieve the full definition of a specific schema by its slug. Returns the JSON Schema object including type, properties, and validation rules. Use --mode to switch between draft and live versions.",
	Example: `  # Get a schema by slug (positional)
  suprsend schema get order-placed

  # Get using the flag form
  suprsend schema get --slug order-placed

  # Get the draft version
  suprsend schema get order-placed --mode draft`,
	Args:  cobra.MaximumNArgs(1),
	Annotations: map[string]string{
		"skills:tip:output": "Use `-o json` for machine-readable JSON output, `-o yaml` for YAML.",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		slug := utils.ResolveSlug(cmd, args)
		if slug == "" {
			return clierr.New("slug is required: provide it as a positional argument or via --slug", clierr.CodeInvalidUsage)
		}
		workspace, _ := cmd.Flags().GetString("workspace")
		mode, _ := cmd.Flags().GetString("mode")
		outputType, _ := cmd.Flags().GetString("output")
		if err := utils.ValidateOutputType(outputType, "json", "yaml"); err != nil {
			return err
		}
		mgmntClient := utils.GetSuprSendMgmntClient()
		spinner := utils.NewSpinner("Getting details...")

		schema, err := mgmntClient.GetSchemaBySlug(workspace, slug, mode)
		if err != nil {
			spinner.Stop("")
			log.WithError(err).Errorf("Error getting schema detail")
			return clierr.Wrap(err, clierr.CodeAPIInternal, "")
		}

		spinner.Stop(fmt.Sprintf("Successfully got details for '%s'", slug))

		utils.OutputData(schema, outputType)
		return nil
	},
}

func init() {
	schemaGetCmd.PersistentFlags().StringP("slug", "g", "", "Schema slug")
	schemaGetCmd.PersistentFlags().StringP("mode", "m", "live", "Version mode: draft or live")
	schemaGetCmd.PersistentFlags().StringP("output", "o", "json", "Output format: json or yaml")
	SchemaCmd.AddCommand(schemaGetCmd)
}
