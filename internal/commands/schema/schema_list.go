package schema

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/utils"
	"github.com/yarlson/pin"
)

var schemaListCmd = &cobra.Command{
	Use:   "list",
	Short: "List schemas",
	Long:  `List trigger payload schemas in a workspace with pagination. Returns schema slug, name, and version info. Use --mode to switch between draft and live versions.`,
	Annotations: map[string]string{
		"skills:tip:output": "Use `-o json` for machine-readable JSON output, `-o yaml` for YAML. Default `-o pretty` outputs a human-friendly table.",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")
		mode, _ := cmd.Flags().GetString("mode")

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
		schemas, err := mgmntClient.ListSchema(workspace, limit, offset, mode)
		if err != nil {
			log.WithError(err).Error("Couldn't fetch schemas")
			return err
		}
		if p != nil {
			p.Stop(fmt.Sprintf("Listed %d schemas from %s with offset %d", len(schemas.Results), workspace, offset))
		}

		outputType, _ := cmd.Flags().GetString("output")
		if len(schemas.Results) == 0 && utils.IsOutputPiped() {
			utils.OutputData([]interface{}{}, outputType)
			return nil
		}
		filteredSchemas := filterSchemaData(schemas.Results)
		utils.OutputData(filteredSchemas, outputType)
		return nil
	},
}

func init() {
	schemaListCmd.PersistentFlags().IntP("limit", "l", 20, "Maximum number of schemas to return")
	schemaListCmd.PersistentFlags().Int("offset", 0, "Number of schemas to skip for pagination")
	schemaListCmd.PersistentFlags().StringP("mode", "m", "live", "Version mode: draft or live")
	schemaListCmd.PersistentFlags().StringP("output", "o", "pretty", "Output format: pretty, json, or yaml")

	SchemaCmd.PersistentFlags().StringP("workspace", "w", "staging", "Workspace name (e.g., staging, production)")
	SchemaCmd.PersistentFlags().StringP("service-token", "s", "", "Service token (default: $SUPRSEND_SERVICE_TOKEN)")
	SchemaCmd.AddCommand(schemaListCmd)
}
