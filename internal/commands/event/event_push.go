package event

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/utils"
)

var eventPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push linked events",
	Long:  "Push event definitions from local per-event directories to a workspace in a single bulk request. Reads `event.json` from every subdirectory of the input directory (default `./suprsend/events`); the event name comes from the `name` field inside each file, not from the directory name.",
	Example: `  # Push all events from default directory
  suprsend event push

  # Push from a custom directory
  suprsend event push --dir ./my-events

  # Push an event inline via JSON, linked to the "user-signup" schema
  suprsend event push --json '{"events":[{"name":"user_signed_up","payload_schema":{"schema":"user-signup"}}]}'

  # Dry run: preview what would be pushed without making changes
  suprsend event push --dry-run`,
	Annotations: map[string]string{
		"skills:tip.a-results": "The API saves the valid events and reports each event separately. Each event is reported on stdout as `pushed` or `failed`; rejected events (for example a `payload_schema` that names a schema or `version_no` that does not exist or is not live in the workspace) are listed by name and the command exits 1; the other events are still saved. With `-o json` the result is one JSON object with `items` and `counts`.",
		"skills:tip.b-names":   "The API stores event names in upper case: `Order_Placed` is saved, triggered and pulled as `ORDER_PLACED`.",
	},
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
			// Unlike an empty events directory, an empty or missing "events"
			// array here is a mistake in the payload, so don't treat it as
			// "nothing to push".
			if evs, ok := payload["events"].([]any); !ok || len(evs) == 0 {
				return clierr.New(`--json must be an object with a non-empty "events" array`, clierr.CodeInvalidUsage)
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
		report := utils.NewReport("event push", workspace, dryRun)
		if len(eventsArr) == 0 {
			return report.Finish(nil)
		}

		if dryRun {
			for i := range eventsArr {
				report.Add("events/"+EventName(eventsArr, i), utils.StatusWouldPush, "")
			}
			return report.Finish(nil)
		}

		spinner := utils.NewSpinner("Pushing events...")
		mgmntClient := utils.GetSuprSendMgmntClient()
		records, err := mgmntClient.PushEventsFromPayload(workspace, payload)
		spinner.Stop("")
		if err != nil {
			return report.Finish(clierr.Wrap(err, clierr.CodeAPIInternal, ""))
		}
		for i := range eventsArr {
			name := "events/" + EventName(eventsArr, i)
			if i < len(records) && records[i].Failed() {
				report.Fail(name, errors.New(RecordMessage(records[i])))
			} else {
				report.Add(name, utils.StatusPushed, "")
			}
		}
		return report.Finish(PushFailures(eventsArr, records))
	},
}

func init() {
	eventPushCmd.Flags().StringP("dir", "d", "", "Directory containing per-event subdirectories (default: ./suprsend/events)")
	eventPushCmd.Flags().StringP("json", "j", "", `Events payload as a JSON object with an "events" array, e.g. '{"events":[{"name":"user_signed_up","description":"Signup completed","payload_schema":{"schema":"user-signup","version_no":1}}]}'. payload_schema links the event to an existing schema by slug (version_no is optional); it is not an inline JSON Schema`)
	eventPushCmd.Flags().BoolP("dry-run", "n", false, "Report what would be pushed without making any changes")
	EventCmd.AddCommand(eventPushCmd)
}
