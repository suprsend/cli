package event

import "github.com/spf13/cobra"

var EventCmd = &cobra.Command{
	Use:   "event",
	Short: "Manage events",
	Long:  "Manage events. Subcommands let you list events, pull event definitions to local files, and push event-schema mappings to a workspace.",
	Example: `  suprsend event get --output json
  suprsend event pull --dir ./suprsend/events`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
