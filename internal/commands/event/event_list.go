package event

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/utils"
)

var eventListCmd = &cobra.Command{
	Use:   "list",
	Short: "List events",
	Long:  "List all events in a workspace with pagination. Returns event names and their linked schema information.",
	Annotations: map[string]string{
		"skills:tip:output": "Use `-o json` for machine-readable JSON output, `-o yaml` for YAML. Default `-o pretty` outputs a human-friendly table.",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		spinner := utils.NewSpinner("Loading...")
		workspace, _ := cmd.Flags().GetString("workspace")
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")
		mgmntClient := utils.GetSuprSendMgmntClient()
		events, err := mgmntClient.ListEvents(workspace, limit, offset)
		if err != nil {
			log.WithError(err).Error("Couldn't fetch events")
			return clierr.Wrap(err, clierr.CodeAPIInternal, "")
		}

		spinner.Stop(fmt.Sprintf("Showing %d events out of %d from workspace %s\n", len(events.Results), events.Meta.Count, workspace))
		outputType, _ := cmd.Flags().GetString("output")
		if err := utils.ValidateOutputType(outputType, "pretty", "json", "yaml"); err != nil {
			return err
		}
		utils.OutputData(events.Results, outputType)
		return nil
	},
}

func init() {
	eventListCmd.PersistentFlags().IntP("limit", "l", 20, "Maximum number of events to return")
	eventListCmd.PersistentFlags().Int("offset", 0, "Number of events to skip for pagination")
	eventListCmd.PersistentFlags().StringP("output", "o", "pretty", "Output format: pretty, json, or yaml")
	EventCmd.PersistentFlags().StringP("workspace", "w", "staging", "Workspace name (e.g., staging, production)")
	EventCmd.PersistentFlags().StringP("service-token", "s", "", "Service token (default: $SUPRSEND_SERVICE_TOKEN)")
	EventCmd.AddCommand(eventListCmd)
}
