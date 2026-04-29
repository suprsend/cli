package translation

import "github.com/spf13/cobra"

var TranslationCmd = &cobra.Command{
	Use:   "translation",
	Short: "Manage Translations",
	Long:  "Manage template translations. Subcommands let you list, pull, push, and commit translations for notification templates.",
	Example: `  suprsend translation list
  suprsend translation get --output json
  suprsend translation pull --dir ./suprsend/translations
  suprsend translation push --commit`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}
