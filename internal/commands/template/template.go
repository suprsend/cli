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
	Long:  `Manage notification templates. Templates define the content and structure of notifications across channels (email, SMS, push, in-app, etc.). Subcommands let you list, get details, pull to local files, push from local files, and commit templates.`,
	Example: `  suprsend template list
  suprsend template get welcome-email
  suprsend template pull --dir ./suprsend/templates
  suprsend template push welcome-email --commit`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}
