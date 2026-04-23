package translation

import (
	"context"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/utils"
	"github.com/yarlson/pin"
)

var translationCommitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Commit translation",
	Long:  "Promote template translation changes from draft to live mode. Finalizes all pending translation changes in the workspace.",
	Run: func(cmd *cobra.Command, args []string) {
		workspace, _ := cmd.Flags().GetString("workspace")
		commitMessage, _ := cmd.Flags().GetString("commit-message")
		mgmntClient := utils.GetSuprSendMgmntClient()
		var p *pin.Pin
		if !utils.IsOutputPiped() {
			p = pin.New("Committing translation...",
				pin.WithSpinnerColor(pin.ColorCyan),
				pin.WithTextColor(pin.ColorYellow),
			)
			cancel := p.Start(context.Background())
			defer cancel()
		}
		err := mgmntClient.FinalizeTranslation(workspace, commitMessage)
		if err != nil {
			log.Errorf("%s", err)
			return
		}
		if p != nil {
			p.Stop(fmt.Sprintf("Successfully committed translation '%s'", commitMessage))
		} else {
			fmt.Fprintf(os.Stdout, "Successfully committed translation '%s'\n", commitMessage)
		}
	},
}

func init() {
	translationCommitCmd.Flags().StringP("commit-message", "m", "", "Message describing the changes being committed")
	TranslationCmd.AddCommand(translationCommitCmd)
}
