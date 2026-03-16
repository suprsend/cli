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
	Long:  `Manage workflows`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
