package category

import (
	"context"
	"fmt"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/commands/category/translation"
	"github.com/suprsend/cli/internal/utils"
	"github.com/yarlson/pin"
)

var categoryCommitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Commit categories",
	Long:  "Commit categories to a workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		commitMsg, _ := cmd.Flags().GetString("commit-message")
		dir, _ := cmd.Flags().GetString("dir")

		// Determine the category directory
		categoryDir := dir
		if categoryDir == "" {
			categoryDir = filepath.Join(".", "suprsend", "category")
		}

		// Append "translation" to the category directory
		translationDir := filepath.Join(categoryDir, "translation")

		var p *pin.Pin
		if !utils.IsOutputPiped() {
			p = pin.New("Loading...",
				pin.WithSpinnerColor(pin.ColorCyan),
				pin.WithTextColor(pin.ColorYellow),
			)
			cancel := p.Start(context.Background())
			defer cancel()
		}

		translation.PushTranslations(workspace, "", translationDir)

		mgmntClient := utils.GetSuprSendMgmntClient()
		err := mgmntClient.FinalizeCategories(workspace, commitMsg)
		if err != nil {
			log.WithError(err).Error("Couldn't commit categories")
			return err
		}
		if p != nil {
			p.Stop(fmt.Sprintf("Committed categories to %s", workspace))
		}
		return nil
	},
}

func init() {
	categoryCommitCmd.Flags().StringP("dir", "d", "", "Output directory for categories (default: ./suprsend/category)")
	categoryCommitCmd.PersistentFlags().String("commit-message", "", "Commit message")
	CategoryCmd.AddCommand(categoryCommitCmd)
}
