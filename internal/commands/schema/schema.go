package schema

import "github.com/spf13/cobra"

var SchemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Manage trigger payload schemas",
	Long:  `Manage trigger payload schemas`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
