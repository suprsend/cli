package event

import (
	"context"
	"encoding/json"
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
	Long:  "Push event-to-schema mappings from a local event_schema_mapping.json file to a workspace. Reads the mapping file from the specified directory.",
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		path, _ := cmd.Flags().GetString("dir")
		jsonPayload, _ := cmd.Flags().GetString("json")

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

		var err error
		if jsonPayload != "" {
			var events map[string]any
			if err = json.Unmarshal([]byte(jsonPayload), &events); err != nil {
				return fmt.Errorf("failed to parse --json payload: %w", err)
			}
			err = mgmntClient.PushEventsFromPayload(workspace, events)
		} else {
			if path == "" {
				path = filepath.Join(".", "suprsend", "event", "event_schema_mapping.json")
			} else {
				path = filepath.Join(path, "event_schema_mapping.json")
			}
			err = mgmntClient.PushEvents(workspace, path)
		}

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
	eventPushCmd.Flags().StringP("dir", "d", "", "Directory containing event files (default: ./suprsend/event)")
	eventPushCmd.Flags().StringP("json", "j", "", `Events payload as a JSON object with an "events" array, matching the format produced by pull, e.g. '{"events":[{"name":"user_signed_up","description":"...","payload_schema":{...}}]}'`)
	EventCmd.AddCommand(eventPushCmd)
}
