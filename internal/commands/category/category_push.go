package category

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/commands/category/translation"
	"github.com/suprsend/cli/internal/utils"
	"github.com/yarlson/pin"
)

var categoryPushCmd = &cobra.Command{
	Use:   "push",
	Long:  "Push categories to a workspace",
	Short: "Push categories to a workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		path, _ := cmd.Flags().GetString("dir")
		commit, _ := cmd.Flags().GetString("commit")
		commitMessage, _ := cmd.Flags().GetString("commit-message")

		translationDir := path
		if translationDir == "" {
			translationDir = filepath.Join(".", "suprsend", "category")
		}
		translationDir = filepath.Join(translationDir, "translation")

		if path == "" {
			path = filepath.Join(".", "suprsend", "category", "categories_preferences.json")
		} else {
			path = filepath.Join(path, "categories_preferences.json")
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			log.Errorf("Directory %s does not exist", path)
			return err
		}

		categories, err := ReadFromFile(path)
		if err != nil {
			log.WithError(err).Error("Couldn't read categories from file")
			return err
		}

		var p *pin.Pin
		if !utils.IsOutputPiped() {
			p = pin.New("Pushing categories...",
				pin.WithSpinnerColor(pin.ColorCyan),
				pin.WithTextColor(pin.ColorYellow),
			)
			cancel := p.Start(context.Background())
			defer cancel()
		}

		if commit == "true" {
			translation.PushTranslations(workspace, "", translationDir)
		}

		mgmnt_client := utils.GetSuprSendMgmntClient()
		err = mgmnt_client.PushCategories(workspace, categories, commit, commitMessage)
		if err != nil {
			log.WithError(err).Error("Couldn't push categories")
			return err
		}
		if p != nil {
			p.Stop(fmt.Sprintf("Pushed categories to %s", workspace))
		}
		return nil
	},
}

func init() {
	categoryPushCmd.Flags().StringP("dir", "d", "", "Output directory for categories (default: ./suprsend/category/)")
	categoryPushCmd.PersistentFlags().StringP("commit", "c", "true", "Commit the categories ")
	categoryPushCmd.PersistentFlags().StringP("commit-message", "m", "", "Commit message for the categories")
	CategoryCmd.AddCommand(categoryPushCmd)
}
