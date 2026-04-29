package translation

import (
	"github.com/spf13/cobra"
)

var TranslationCmd = &cobra.Command{
	Use:   "translation",
	Short: "Manage preference category translations",
	Long:  "Manage preference category translations. List available translation locales, pull translations from a workspace to local files, or push local translation files back to a workspace.",
	Example: `  suprsend category translation list
  suprsend category translation pull --dir ./suprsend/categories
  suprsend category translation push --locale es`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}
