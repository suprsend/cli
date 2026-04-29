/*
Copyright © 2025 SuprSend
*/
package workflow

import (
	"github.com/spf13/cobra"
)

// workflowCmd represents the workflow command
var WorkflowCmd = &cobra.Command{
	Use:   "workflow",
	Short: "Manage workflows",
	Long:  `Manage workflows. Subcommands let you list, get details, pull to local files, push from local files, and enable/disable workflows in a workspace.`,
	Example: `  suprsend workflow list
  suprsend workflow get welcome
  suprsend workflow pull --dir ./suprsend/workflows
  suprsend workflow push welcome --commit`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
