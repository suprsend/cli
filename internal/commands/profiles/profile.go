package profiles

import (
	"github.com/spf13/cobra"
)

var ProfileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage Profile",
	Long:  "Manage Profile and store credentials. Only useful if you have a BYOC/self-hosted SuprSend instance or if you want to manage multiple accounts. Not required for moving assets between workspaces in the same account. Not required for SaaS users.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
