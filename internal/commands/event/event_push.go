package event

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/utils"
	"github.com/yarlson/pin"
)

var eventPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push linked events",
	Long:  "Push linked events in schemas",
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		path, _ := cmd.Flags().GetString("dir")
		if path == "" {
			path = filepath.Join(".", "suprsend", "event", "event_schema_mapping.json")
		} else {
			path = filepath.Join(path, "event_schema_mapping.json")
		}

		var p *pin.Pin
		if !utils.IsOutputPiped() {
			p = pin.New("Pushing events...",
				pin.WithSpinnerColor(pin.ColorCyan),
				pin.WithTextColor(pin.ColorYellow),
			)
			cancel := p.Start(context.Background())
			defer cancel()
		}

		mgmntClient := utils.GetSuprSendMgmntClient()
		err := mgmntClient.PushEvents(workspace, path)
		if err != nil {
			if p != nil {
				p.Stop("")
			}
			log.WithError(err).Error("Failed to push events")
			return err
		}
		if p != nil {
			p.Stop("Successfully pushed events")
		} else {
			fmt.Fprintf(os.Stdout, "Successfully pushed events\n")
		}
		return nil
	},
}

func init() {
	eventPushCmd.Flags().StringP("dir", "d", "", "Directory to push events from (default: ./suprsend/event)")
	EventCmd.AddCommand(eventPushCmd)
}
