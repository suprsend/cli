package translation

import "github.com/spf13/cobra"

var TranslationCmd = &cobra.Command{
	Use:   "translation",
	Short: "Manage Translations",
	Long:  "Manage template translations. Subcommands let you list, pull, push, and commit translations for notification templates.",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}
