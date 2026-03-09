package event

import "github.com/spf13/cobra"

var EventCmd = &cobra.Command{
	Use:   "event",
	Short: "Manage events",
	Long:  "Manage events",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
