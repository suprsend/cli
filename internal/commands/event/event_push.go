package event

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/utils"
)

var eventPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push linked events",
	Long:  "Push event definitions from local per-event directories to a workspace. Reads from events/<name>/event.json files in the specified directory.",
	Example: `  # Push all events from default directory
  suprsend event push

  # Push from a custom directory
  suprsend event push --dir ./my-events

  # Push events inline via JSON
  suprsend event push --json '{"events":[{"name":"user_signed_up","payload_schema":{...}}]}'

  # Dry run: preview what would be pushed without making changes
  suprsend event push --dry-run`,
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		dir, _ := cmd.Flags().GetString("dir")
		jsonPayload, _ := cmd.Flags().GetString("json")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		// Resolve and validate the events payload BEFORE calling the API,
		// so dry-run still surfaces parse / file-not-found errors.
		var payload map[string]any
		if jsonPayload != "" {
			if err := json.Unmarshal([]byte(jsonPayload), &payload); err != nil {
				return clierr.Wrap(err, clierr.CodeFileParseFailed, "")
			}
		} else {
			if dir == "" {
				dir = filepath.Join(".", "suprsend", "events")
			}
			if _, statErr := os.Stat(dir); os.IsNotExist(statErr) {
				return clierr.New(fmt.Sprintf("events directory not found: %s", dir), clierr.CodeFileNotFound)
			}
			events, readErr := ReadEventsFromDir(dir)
			if readErr != nil {
				return clierr.Wrap(readErr, clierr.CodeFileParseFailed, "")
			}
			payload = map[string]any{"events": events}
		}

		eventsArr, _ := payload["events"].([]any)

		if dryRun {
			log.Infof("DRY RUN: would push %d event(s) to %s", len(eventsArr), workspace)
			for _, ev := range eventsArr {
				if obj, ok := ev.(map[string]any); ok {
					name, _ := obj["name"].(string)
					if name != "" {
						log.Infof("  - %s", name)
					}
				}
			}
			return nil
		}

		// Single bulk POST: the management endpoint accepts an "events"
		// array and the entire batch is one transaction. The earlier
		// per-event loop (commit f844a95) was reverted because it
		// scaled latency linearly with workspace size — large workspaces
		// (hundreds of events) hit an unacceptable cliff.
		spinner := utils.NewSpinner("Pushing events...")
		mgmntClient := utils.GetSuprSendMgmntClient()
		if err := mgmntClient.PushEventsFromPayload(workspace, payload); err != nil {
			spinner.Stop("")
			log.WithError(err).Error("Failed to push events")
			return clierr.Wrap(err, clierr.CodeAPIInternal, "")
		}
		spinner.Stop(fmt.Sprintf("Pushed %d event(s) to %s", len(eventsArr), workspace))
		return nil
	},
}

func init() {
	eventPushCmd.Flags().StringP("dir", "d", "", "Directory containing per-event subdirectories (default: ./suprsend/events)")
	eventPushCmd.Flags().StringP("json", "j", "", `Events payload as a JSON object with an "events" array, e.g. '{"events":[{"name":"user_signed_up","description":"...","payload_schema":{...}}]}'`)
	eventPushCmd.Flags().BoolP("dry-run", "n", false, "Print what would be pushed without making any changes")
	EventCmd.AddCommand(eventPushCmd)
}
