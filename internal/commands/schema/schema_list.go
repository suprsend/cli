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
	Long:  `List schemas in a workspace`,
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
	schemaListCmd.PersistentFlags().IntP("limit", "l", 20, "Limit the number of schemas to list")
	schemaListCmd.PersistentFlags().IntP("offset", "f", 0, "Offset the number of schemas to list (default: 0)")
	schemaListCmd.PersistentFlags().StringP("mode", "m", "live", "Mode of schemas to list (draft, live), default: live")
	schemaListCmd.PersistentFlags().StringP("output", "o", "pretty", "Output Style (pretty, yaml, json)")

	SchemaCmd.PersistentFlags().StringP("workspace", "w", "staging", "Workspace to use the schemas from")
	SchemaCmd.PersistentFlags().StringP("service-token", "s", "", "Service token (default: $SUPRSEND_SERVICE_TOKEN)")
	SchemaCmd.AddCommand(schemaListCmd)
}
