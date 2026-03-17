package category

import (
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/commands/category/translation"
)

var CategoryCmd = &cobra.Command{
	Use:   "category",
	Short: "Manage preference categories",
	Long:  "Manage notification preference categories. Categories organize notification preferences into a hierarchy of root categories, sections, and individual preference items.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	CategoryCmd.PersistentFlags().StringP("workspace", "w", "staging", "Workspace name (e.g., staging, production)")
	CategoryCmd.AddCommand(translation.TranslationCmd)
}
