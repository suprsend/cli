package event

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/utils"
)

var eventGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get events",
	Long:  "Retrieve all events and their schema mappings from a workspace. Returns event definitions including names, descriptions, and payload schemas.",
	Example: `  # Get all events (JSON output recommended for event details)
  suprsend event get

  # Get with JSON output
  suprsend event get --output json

  # Get from a specific workspace
  suprsend event get --workspace production`,
	Annotations: map[string]string{
		"skills:tip:output": "Use `-o json` for machine-readable JSON output, `-o yaml` for YAML.",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		outputType, _ := cmd.Flags().GetString("output")
		if err := utils.ValidateOutputType(outputType, "json", "yaml"); err != nil {
			return err
		}
		mgmntClient := utils.GetSuprSendMgmntClient()
		spinner := utils.NewSpinner("Getting events...")

		eventsResp, err := mgmntClient.GetEvents(workspace)
		if err != nil {
			spinner.Stop("")
			log.WithError(err).Errorf("Error getting events")
			return clierr.Wrap(err, clierr.CodeAPIInternal, "check your service token and workspace")
		}

		spinner.Stop(fmt.Sprintf("Successfully got %d event(s)", len(eventsResp.Results)))

		output := map[string]any{"events": eventsResp.Results}
		utils.OutputData(output, outputType)
		return nil
	},
}

func init() {
	eventGetCmd.PersistentFlags().StringP("output", "o", "json", "Output format: json or yaml")
	EventCmd.AddCommand(eventGetCmd)
}
