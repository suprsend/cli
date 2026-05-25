package event

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/utils"
)

var eventGetCmd = &cobra.Command{
	Use:   "get [<slug>]",
	Short: "Get event details",
	Long:  "Retrieve a specific event by its slug. Returns the event definition including name, description, and payload schema.",
	Example: `  # Get a specific event by slug
  suprsend event get ORDER_RECEIVED

  # Get using the flag form
  suprsend event get --slug ORDER_RECEIVED

  # Get from a specific workspace
  suprsend event get --workspace production`,
	Args: cobra.MaximumNArgs(1),
	Annotations: map[string]string{
		"skills:tip:output": "Use `-o json` for machine-readable JSON output, `-o yaml` for YAML.",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		name := utils.ResolveSlug(cmd, args)
		workspace, _ := cmd.Flags().GetString("workspace")
		outputType, _ := cmd.Flags().GetString("output")
		if err := utils.ValidateOutputType(outputType, "json", "yaml"); err != nil {
			return err
		}
		mgmntClient := utils.GetSuprSendMgmntClient()

		if name == "" {
			return clierr.New("slug is required: provide it as a positional argument or via --slug", clierr.CodeInvalidUsage)
		}

		spinner := utils.NewSpinner("Getting details...")
		event, err := mgmntClient.GetEventDetail(cmd.Context(), workspace, name)
		if err != nil {
			spinner.Stop("")
			log.WithError(err).Errorf("Error getting event detail")
			return clierr.Wrap(err, clierr.CodeAPIInternal, "")
		}
		spinner.Stop(fmt.Sprintf("Successfully got details for '%s'", name))
		utils.OutputData(event, outputType)
		return nil
	},
}

func init() {
	eventGetCmd.PersistentFlags().StringP("slug", "g", "", "Event slug")
	eventGetCmd.PersistentFlags().StringP("output", "o", "json", "Output format: json or yaml")
	EventCmd.AddCommand(eventGetCmd)
}
