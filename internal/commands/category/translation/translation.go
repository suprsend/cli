package translation

import (
	"github.com/spf13/cobra"
)

var TranslationCmd = &cobra.Command{
	Use:   "translation",
	Short: "Manage preference category translations",
	Long:  "Manage preference category translations. List available translation locales, pull translations from a workspace to local files, or push local translation files back to a workspace.",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}
