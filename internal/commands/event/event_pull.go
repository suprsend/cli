package event

import (
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/utils"
)

var eventPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull events from workspace to local directory",
	Long:  "Download event definitions from a workspace to local files. Saves each event as `<dir>/<event_dir>/event.json` (default dir `./suprsend/events`). `<event_dir>` is the event name made filesystem-safe: characters other than letters, digits, `.`, `_` and `-` become `_`, leading and trailing non-alphanumeric characters are dropped (a leading `$` is kept), and the result is capped at 128 characters. If two event names map to the same directory name (compared case-insensitively), only the first is written and the other is skipped with a warning.",
	Example: `  # Pull all events to default directory (suprsend/events/)
  suprsend event pull

  # Pull to a custom directory
  suprsend event pull --dir ./my-events

  # Pull from the production workspace
  suprsend event pull --workspace production`,
	Annotations: map[string]string{
		"skills:tip.a-overwrite": "Pull overwrites `<dir>/<event_dir>/event.json` for every pulled event without asking, so commit or stash local edits first. `--force` does not affect overwriting: it only skips the output-directory prompt shown when `--dir` is omitted and `./suprsend/events` doesn't exist. In non-interactive shells pass `--dir` or `--force`; otherwise the command exits 64 (invalid usage) without pulling anything.",
		"skills:tip.b-dirname":   "The directory name is a sanitized form of the event name (e.g. `Order Placed` → `Order_Placed`), so don't derive the event name from it: the real name is the `name` field inside `event.json`.",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		dirPath, _ := cmd.Flags().GetString("dir")
		force, _ := cmd.Flags().GetBool("force")
		if dirPath == "" {
			dirPath = filepath.Join(".", "suprsend", "events")
			if _, err := os.Stat(dirPath); os.IsNotExist(err) {
				if force {
					log.Infof("Using default directory: %s", dirPath)
				} else {
					od, success := promptForOutputDirectory()
					if !success {
						return clierr.New("output directory is required; pass --dir, or --force to use the default", clierr.CodeInvalidUsage)
					}
					dirPath = od
				}
			}
			if dirPath == "" {
				log.Info("No output directory specified. Exiting.")
				return nil
			}
		}
		spinner := utils.NewSpinner("Loading...")

		mgmntClient := utils.GetSuprSendMgmntClient()
		eventsResp, err := mgmntClient.GetEvents(workspace)
		if err != nil {
			log.Errorf("Failed to get events: %v", err)
			return clierr.Wrap(err, clierr.CodeAPIInternal, "")
		}
		spinner.Stop(fmt.Sprintf("Pulled %d events", len(eventsResp.Results)))

		stats, err := WriteEventsToFiles(eventsResp, dirPath)
		if err != nil {
			log.Errorf("Failed to save events: %v", err)
			return clierr.Wrap(err, clierr.CodeFileParseFailed, "")
		}
		if stats.Failed > 0 {
			return clierr.Failures(fmt.Sprintf("%d event(s) failed to pull", stats.Failed), stats.Errors, clierr.CodeAPIInternal)
		}
		return nil
	},
}

func init() {
	eventPullCmd.Flags().StringP("dir", "d", "", "Directory to save event files to (default: ./suprsend/events)")
	eventPullCmd.PersistentFlags().BoolP("force", "F", false, "Skip directory confirmation prompt, use default path")
	EventCmd.AddCommand(eventPullCmd)
}
