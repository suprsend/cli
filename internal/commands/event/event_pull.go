package event

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/utils"
	"github.com/yarlson/pin"
)

var eventPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull events from workspace to local directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		dirPath, _ := cmd.Flags().GetString("dir")
		force, _ := cmd.Flags().GetBool("force")
		if dirPath == "" {
			dirPath = filepath.Join(".", "suprsend", "event")
			if _, err := os.Stat(dirPath); os.IsNotExist(err) {
				if force {
					fmt.Fprintf(os.Stdout, "Using default directory: %s\n", dirPath)
				} else {
					dirPath = promptForOutputDirectory()
				}
			}
			if dirPath == "" {
				fmt.Fprintf(os.Stdout, "No output directory specified. Exiting \n")
				return nil
			}
		}
		var p *pin.Pin
		if !utils.IsOutputPiped() {
			p = pin.New("Loading...",
				pin.WithSpinnerColor(pin.ColorCyan),
				pin.WithTextColor(pin.ColorYellow),
			)
			cancel := p.Start(context.Background())
			defer cancel()
		}

		mgmntClient := utils.GetSuprSendMgmntClient()
		eventsResp, err := mgmntClient.GetEvents(workspace)
		if err != nil {
			fmt.Fprintf(os.Stdout, "Error: Failed to get events: %v\n", err)
			return err
		}
		if p != nil {
			p.Stop(fmt.Sprintf("Pulled %d events", len(eventsResp.Results)))
		}

		_, err = WriteEventsToFiles(eventsResp, dirPath)
		if err != nil {
			fmt.Fprintf(os.Stdout, "Error: Failed to save events: %v\n", err)
			return err
		}
		return nil
	},
}

func init() {
	eventPullCmd.Flags().StringP("dir", "d", "", "Directory to pull events to (default: ./suprsend/event)")
	eventPullCmd.PersistentFlags().BoolP("force", "f", false, "Force using default directory without prompting")
	EventCmd.AddCommand(eventPullCmd)
}
