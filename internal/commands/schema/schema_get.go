package schema

import (
	"context"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/utils"
	"github.com/yarlson/pin"
)

var schemaGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get schema details",
	Long:  "Get schema details for a specific schema. Example: suprsend schema get --slug <slug>",
	RunE: func(cmd *cobra.Command, args []string) error {
		slug, _ := cmd.Flags().GetString("slug")
		if slug == "" {
			log.Error("Schema slug is required. Example: suprsend schema get --slug <slug>")
			return fmt.Errorf("schema slug is required. Example: suprsend schema get --slug <slug>")
		}
		workspace, _ := cmd.Flags().GetString("workspace")
		mode, _ := cmd.Flags().GetString("mode")
		outputType, _ := cmd.Flags().GetString("output")
		mgmntClient := utils.GetSuprSendMgmntClient()
		var p *pin.Pin
		if !utils.IsOutputPiped() {
			p = pin.New("Getting details...",
				pin.WithSpinnerColor(pin.ColorCyan),
				pin.WithTextColor(pin.ColorYellow),
			)
			cancel := p.Start(context.Background())
			defer cancel()
		}

		schema, err := mgmntClient.GetSchemaBySlug(workspace, slug, mode)
		if err != nil {
			log.WithError(err).Errorf("Error getting schema detail")
			return err
		}
		utils.OutputData(schema, outputType)
		if p != nil {
			p.Stop(fmt.Sprintf("Successfully got details for '%s'", slug))
		} else {
			fmt.Fprintf(os.Stdout, "Successfully got details for '%s'", slug)
		}
		return nil
	},
}

func init() {
	schemaGetCmd.PersistentFlags().StringP("slug", "g", "", "Slug of the schema to get")
	schemaGetCmd.PersistentFlags().String("mode", "live", "mode to fetch schema from.")
	schemaGetCmd.PersistentFlags().StringP("output", "o", "json", "Output format (json, yaml)")
	SchemaCmd.AddCommand(schemaGetCmd)
}
