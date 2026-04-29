package schema

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/utils"
)

var schemaCommitCmd = &cobra.Command{
	Use:   "commit [<slug>]",
	Short: "Commit schema from draft to live",
	Long:  `Promote a schema from draft to live mode. Pass the schema slug as a positional argument or via --slug. Once committed, the schema changes become active immediately.`,
	Example: `  # Commit a schema to live (positional slug)
  suprsend schema commit order-placed

  # Commit using the flag form
  suprsend schema commit --slug order-placed

  # Commit in the production workspace
  suprsend schema commit order-placed --workspace production`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		slug := utils.ResolveSlug(cmd, args)
		if slug == "" {
			return clierr.New("schema slug is required: provide it as a positional argument or via --slug", clierr.CodeInvalidUsage)
		}

		workspace, _ := cmd.Flags().GetString("workspace")
		commitMessage, _ := cmd.Flags().GetString("commit-message")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		if dryRun {
			log.Infof("DRY RUN: would commit schema '%s' to live", slug)
			return nil
		}

		force, _ := cmd.Flags().GetBool("force")
		if !force {
			msg := fmt.Sprintf("This will promote schema '%s' to live in workspace \"%s\". Continue?", slug, workspace)
			confirmed, err := utils.ConfirmDestructiveAction(msg)
			if err != nil || !confirmed {
				log.Info("Aborted.")
				return nil
			}
		}

		mgmntClient := utils.GetSuprSendMgmntClient()
		spinner := utils.NewSpinner("Committing schema...")

		err := mgmntClient.FinalizeSchema(workspace, slug, commitMessage)
		if err != nil {
			log.Error(err.Error())
			return clierr.Wrap(err, clierr.CodeAPIInternal, "")
		}

		spinner.Stop(fmt.Sprintf("Successfully committed schema '%s' to live mode", slug))
		return nil
	},
}

func init() {
	schemaCommitCmd.Flags().StringP("slug", "g", "", "Schema slug")
	schemaCommitCmd.Flags().String("commit-message", "", "Message describing the changes being committed")
	schemaCommitCmd.Flags().BoolP("dry-run", "n", false, "Print what would be committed without making any changes")
	schemaCommitCmd.Flags().BoolP("force", "F", false, "Skip confirmation prompt")
	SchemaCmd.AddCommand(schemaCommitCmd)
}
