package event

import (
	"context"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/utils"
	"github.com/yarlson/pin"
)

var eventGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get events",
	Long:  "Get all events for the workspace. Example: suprsend event get",
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		outputType, _ := cmd.Flags().GetString("output")
		mgmntClient := utils.GetSuprSendMgmntClient()
		var p *pin.Pin
		if !utils.IsOutputPiped() {
			p = pin.New("Getting events...",
				pin.WithSpinnerColor(pin.ColorCyan),
				pin.WithTextColor(pin.ColorYellow),
			)
			cancel := p.Start(context.Background())
			defer cancel()
		}

		eventsResp, err := mgmntClient.GetEvents(workspace)
		if err != nil {
			log.WithError(err).Errorf("Error getting events")
			return err
		}
		output := map[string]any{"events": eventsResp.Results}
		utils.OutputData(output, outputType)
		if p != nil {
			p.Stop(fmt.Sprintf("Successfully got %d event(s)", len(eventsResp.Results)))
		} else {
			fmt.Fprintf(os.Stdout, "Successfully got %d event(s)", len(eventsResp.Results))
		}
		return nil
	},
}

func init() {
	eventGetCmd.PersistentFlags().StringP("output", "o", "json", "Output format (json, yaml)")
	EventCmd.AddCommand(eventGetCmd)
}
