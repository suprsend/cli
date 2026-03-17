package schema

import "github.com/spf13/cobra"

var SchemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Manage trigger payload schemas",
	Long:  `Manage trigger payload schemas. Schemas define the JSON structure for workflow and event trigger payloads. Subcommands let you list, pull, push, and commit schemas.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
