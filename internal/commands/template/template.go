/*
Copyright © 2025 SuprSend
*/
package template

import (
	"github.com/spf13/cobra"
)

// TemplateCmd represents the template command
var TemplateCmd = &cobra.Command{
	Use:   "template",
	Short: "Manage templates",
	Long:  `Manage templates`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}
