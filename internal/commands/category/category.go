package category

import (
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/commands/category/translation"
)

var CategoryCmd = &cobra.Command{
	Use:   "category",
	Short: "Manage preference categories",
	Long:  "Manage preference categories",
}

func init() {
	CategoryCmd.PersistentFlags().StringP("workspace", "w", "staging", "Workspace to push categories to")
	CategoryCmd.AddCommand(translation.TranslationCmd)
}
