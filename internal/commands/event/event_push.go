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

// EventPushStats tracks per-event success/failure for the push summary,
// mirroring the shape used by schema push so output reads consistently.
type EventPushStats struct {
	Total   int
	Success int
	Failed  int
	Errors  []string
}

var eventPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push linked events",
	Long:  "Push event definitions from local per-event directories to a workspace. Reads from events/<name>/event.json files in the specified directory. Each event is pushed independently — a failure on one doesn't abort the rest.",
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

		// Push each event independently, mirroring schema push's pattern:
		// per-item spinner, log on success/failure, end-of-run summary.
		// The management API only exposes a bulk endpoint so each iteration
		// sends a single-element {"events": [...]} payload — N small POSTs
		// instead of one big POST. Trade-off: ~Nx latency, but per-event
		// failure visibility, which is what users want when something
		// goes wrong on push #7 of 12.
		mgmntClient := utils.GetSuprSendMgmntClient()
		stats := &EventPushStats{Total: len(eventsArr), Errors: []string{}}
		var spinner *utils.Spinner

		for _, ev := range eventsArr {
			obj, ok := ev.(map[string]any)
			if !ok {
				stats.Failed++
				stats.Errors = append(stats.Errors, "event entry was not an object")
				continue
			}
			name, _ := obj["name"].(string)
			if name == "" {
				stats.Failed++
				stats.Errors = append(stats.Errors, "event entry missing required 'name' field")
				continue
			}

			spinner = utils.NewSpinner(fmt.Sprintf("Pushing %s...", name))
			err := mgmntClient.PushEventsFromPayload(workspace, map[string]any{"events": []any{obj}})
			if err != nil {
				spinner.Stop("")
				log.WithError(err).Errorf("events/%s: failed to push", name)
				stats.Failed++
				stats.Errors = append(stats.Errors, fmt.Sprintf("events/%s: failed to push: %v", name, err))
				continue
			}
			stats.Success++
			spinner.Stop(fmt.Sprintf("Pushed event: %s", name))
		}

		log.Info("=== Event Push Summary ===")
		log.Infof("Total events processed: %d", stats.Total)
		log.Infof("Successfully pushed: %d", stats.Success)
		log.Infof("Failed to push: %d", stats.Failed)

		if stats.Failed > 0 {
			log.Info("Failed events:")
			for _, errorMsg := range stats.Errors {
				log.Infof("  - %s", errorMsg)
			}
			return clierr.New(fmt.Sprintf("%d event(s) failed to push", stats.Failed), clierr.CodeAPIInternal)
		}
		return nil
	},
}

func init() {
	eventPushCmd.Flags().StringP("dir", "d", "", "Directory containing per-event subdirectories (default: ./suprsend/events)")
	eventPushCmd.Flags().StringP("json", "j", "", `Events payload as a JSON object with an "events" array, e.g. '{"events":[{"name":"user_signed_up","description":"...","payload_schema":{...}}]}'`)
	eventPushCmd.Flags().BoolP("dry-run", "n", false, "Print what would be pushed without making any changes")
	EventCmd.AddCommand(eventPushCmd)
}
