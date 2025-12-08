package translation

import "github.com/spf13/cobra"

var TranslationCmd = &cobra.Command{
	Use:   "translation",
	Short: "Manage Translations",
	Long:  "Manage Translation",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}
