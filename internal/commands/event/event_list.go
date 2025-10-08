package event

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/utils"
	"github.com/yarlson/pin"
)

var eventListCmd = &cobra.Command{
	Use:   "list",
	Short: "List events",
	Long:  "List all events",
	RunE: func(cmd *cobra.Command, args []string) error {
		var p *pin.Pin
		if !utils.IsOutputPiped() {
			p = pin.New("Loading...",
				pin.WithSpinnerColor(pin.ColorCyan),
				pin.WithTextColor(pin.ColorYellow),
			)
			cancel := p.Start(context.Background())
			defer cancel()
		}
		workspace, _ := cmd.Flags().GetString("workspace")
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")
		mgmntClient := utils.GetSuprSendMgmntClient()
		events, err := mgmntClient.ListEvents(workspace, limit, offset)
		if err != nil {
			log.WithError(err).Error("Couldn't fetch events")
			return err
		}

		if p != nil {
			p.Stop(fmt.Sprintf("Showing %d events out of %d from workspace %s\n", len(events.Results), events.Meta.Count, workspace))
		}
		outputType, _ := cmd.Flags().GetString("output")
		utils.OutputData(events.Results, outputType)
		return nil
	},
}

func init() {
	eventListCmd.PersistentFlags().IntP("limit", "l", 20, "Limit the number of events to list.")
	eventListCmd.PersistentFlags().IntP("offset", "f", 0, "Offset into the list of events(default: 0)")
	eventListCmd.PersistentFlags().StringP("output", "o", "pretty", "Output Style (pretty, yaml, json)")
	EventCmd.PersistentFlags().StringP("workspace", "w", "staging", "Workspace to list events from")
	EventCmd.PersistentFlags().StringP("service-token", "s", "", "Service token (default: $SUPRSEND_SERVICE_TOKEN)")
	EventCmd.AddCommand(eventListCmd)
}
