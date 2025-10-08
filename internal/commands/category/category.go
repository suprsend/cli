package category

import "github.com/spf13/cobra"

var CategoryCmd = &cobra.Command{
	Use:   "category",
	Short: "Manage preference categories",
	Long:  "Manage preference categories",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.Help()
		return nil
	},
}

func init() {
	CategoryCmd.PersistentFlags().StringP("workspace", "w", "staging", "Workspace to push categories to")
}
