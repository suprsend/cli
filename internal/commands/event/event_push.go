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
	Long:  "Push event definitions from local per-event directories to a workspace. Reads from events/<name>/event.json files in the specified directory.",
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		dir, _ := cmd.Flags().GetString("dir")
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
			if dir == "" {
				dir = filepath.Join(".", "suprsend", "events")
			}
			if _, statErr := os.Stat(dir); os.IsNotExist(statErr) {
				return fmt.Errorf("events directory not found: %s", dir)
			}
			events, readErr := ReadEventsFromDir(dir)
			if readErr != nil {
				return readErr
			}
			err = mgmntClient.PushEventsFromPayload(workspace, map[string]any{"events": events})
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
			log.Info("Successfully pushed events")
		}
		return nil
	},
}

func init() {
	eventPushCmd.Flags().StringP("dir", "d", "", "Directory containing per-event subdirectories (default: ./suprsend/events)")
	eventPushCmd.Flags().StringP("json", "j", "", `Events payload as a JSON object with an "events" array, e.g. '{"events":[{"name":"user_signed_up","description":"...","payload_schema":{...}}]}'`)
	EventCmd.AddCommand(eventPushCmd)
}
