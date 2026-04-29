package workspace

import (
	"github.com/spf13/cobra"
)

var WorkspaceCmd = &cobra.Command{
	Use:   "workspace",
	Short: "Manage workspaces",
	Long:  "Manage SuprSend workspaces. Workspaces isolate notification resources (templates, workflows, categories) and can run in sandbox or live mode.",
	Example: `  suprsend workspace list
  suprsend workspace list --limit 5
  suprsend workspace list --output json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	WorkspaceCmd.PersistentFlags().StringP("service-token", "s", "", "Service token (default: $SUPRSEND_SERVICE_TOKEN)")
	WorkspaceCmd.AddCommand(workspaceListCmd)
}
