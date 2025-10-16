package category

import (
	"context"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/utils"
	"github.com/yarlson/pin"
)

type CategoryTableRow struct {
	RootCategory             string `json:"root_category"`
	Section                  string `json:"section"`
	CategoryName             string `json:"category_name"`
	DefaultPreference        string `json:"default_preference"`
	DefaultMandatoryChannels string `json:"default_mandatory_channels"`
}

var categoryListCmd = &cobra.Command{
	Use:   "list",
	Short: "List categories",
	Long:  "List preferences categories in a workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		mode, _ := cmd.Flags().GetString("mode")

		var p *pin.Pin
		if !utils.IsOutputPiped() {
			p = pin.New("Loading...",
				pin.WithSpinnerColor(pin.ColorCyan),
				pin.WithTextColor(pin.ColorYellow),
			)
			cancel := p.Start(context.Background())
			defer cancel()
		}

		mgmntClient := utils.GetSuprSendMgmntClient()
		categories, err := mgmntClient.ListCategories(workspace, mode)
		if err != nil {
			log.WithError(err).Error("Couldn't fetch categories")
			return err
		}
		outputType, _ := cmd.Flags().GetString("output")

		// Create flattened table rows
		var tableRows []CategoryTableRow
		for _, rootCategory := range categories.RootCategories {
			for _, section := range rootCategory.Sections {
				for _, category := range section.Categories {
					tableRows = append(tableRows, CategoryTableRow{
						RootCategory:             rootCategory.RootCategory,
						Section:                  section.Name,
						CategoryName:             category.Name,
						DefaultPreference:        category.DefaultPreference,
						DefaultMandatoryChannels: strings.Join(category.DefaultMandatoryChannels, ", "),
					})
				}
			}
		}
		if p != nil {
			p.Stop(fmt.Sprintf("Listed %d categories from %s", len(tableRows), workspace))
		}

		if len(tableRows) == 0 && utils.IsOutputPiped() {
			utils.OutputData([]interface{}{}, outputType)
			return nil
		}

		utils.OutputData(tableRows, outputType)
		return nil
	},
}

func init() {
	categoryListCmd.PersistentFlags().StringP("mode", "m", "live", "Mode of preferences to list (draft, live), default: live")
	categoryListCmd.PersistentFlags().StringP("output", "o", "pretty", "Output type (pretty, yaml, json)")
	CategoryCmd.PersistentFlags().StringP("service-token", "s", "", "Service token (default: $SUPRSEND_SERVICE_TOKEN)")
	CategoryCmd.AddCommand(categoryListCmd)
}
