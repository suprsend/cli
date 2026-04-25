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
	Long:  "Download event definitions from a workspace to local files. Saves each event as events/<name>/event.json in the output directory.",
	Example: `  # Pull all events to default directory (suprsend/events/)
  suprsend event pull

  # Pull to a custom directory
  suprsend event pull --dir ./my-events

  # Pull from the production workspace
  suprsend event pull --workspace production`,
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
						return nil
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

		_, err = WriteEventsToFiles(eventsResp, dirPath)
		if err != nil {
			log.Errorf("Failed to save events: %v", err)
			return clierr.Wrap(err, clierr.CodeFileParseFailed, "")
		}
		return nil
	},
}

func init() {
	eventPullCmd.Flags().StringP("dir", "d", "", "Directory to save event files to (default: ./suprsend/events)")
	eventPullCmd.PersistentFlags().BoolP("force", "F", false, "Skip directory confirmation prompt, use default path")
	EventCmd.AddCommand(eventPullCmd)
}
