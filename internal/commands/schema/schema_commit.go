package schema

import (
	"context"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/utils"
	"github.com/yarlson/pin"
)

var schemaCommitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Commit schema from draft to live",
	Long:  `Promote a schema from draft to live mode. Requires a schema slug as a positional argument. Once committed, the schema changes become active immediately.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			log.Error("Schema slug argument is required. Example: suprsend schema commit <slug>")
			return fmt.Errorf("Schema slug argument is required. Example: suprsend schema commit <slug>")
		}
		slug := args[0]

		workspace, _ := cmd.Flags().GetString("workspace")
		commitMessage, _ := cmd.Flags().GetString("commit-message")
		if commitMessage == "" {
			return fmt.Errorf("--commit-message (-m) is required")
		}
		mgmntClient := utils.GetSuprSendMgmntClient()
		var p *pin.Pin
		if !utils.IsOutputPiped() {
			p = pin.New("Committing schema...",
				pin.WithSpinnerColor(pin.ColorCyan),
				pin.WithTextColor(pin.ColorYellow),
			)
			cancel := p.Start(context.Background())
			defer cancel()
		}

		err := mgmntClient.FinalizeSchema(workspace, slug, commitMessage)
		if err != nil {
			log.Error(err.Error())
			return err
		}

		if p != nil {
			p.Stop(fmt.Sprintf("Successfully committed schema '%s' to live mode", slug))
		} else {
			fmt.Fprintf(os.Stdout, "Successfully committed schema '%s' to live mode\n", slug)
		}
		return nil
	},
}

func init() {
	schemaCommitCmd.Flags().StringP("commit-message", "m", "", "Message describing the changes being committed")
	SchemaCmd.AddCommand(schemaCommitCmd)
}
