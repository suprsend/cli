package event

import "github.com/spf13/cobra"

var EventCmd = &cobra.Command{
	Use:   "event",
	Short: "Manage events",
	Long:  "Manage events. Subcommands let you list events, get a single event, pull event definitions to local files, and push event-schema mappings to a workspace.",
	Example: `  suprsend event list --output json
  suprsend event get ORDER_RECEIVED
  suprsend event pull --dir ./suprsend/events`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
