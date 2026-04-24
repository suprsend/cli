/*
Copyright © 2025 SuprSend
*/
package template

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/utils"
	"github.com/yarlson/pin"
)

var templateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List templates for a workspace",
	Long:  `List templates for a workspace`,
	Run: func(cmd *cobra.Command, args []string) {
		var p *pin.Pin
		if !utils.IsOutputPiped() {
			p = pin.New("Loading...",
				pin.WithSpinnerColor(pin.ColorCyan),
				pin.WithTextColor(pin.ColorYellow),
			)
			cancel := p.Start(context.Background())
			defer cancel()
		}
		workspace, _ := cmd.Flags().GetString("workspace")
		mgmntClient := utils.GetSuprSendMgmntClient()

		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")
		mode, _ := cmd.Flags().GetString("mode")

		templates, err := mgmntClient.ListTemplates(workspace, limit, offset, mode)
		if err != nil {
			log.WithError(err).Error("Couldn't fetch templates")
			return
		}

		msg := fmt.Sprintf("Listed %d templates from %s with offset %d", len(templates.Results), workspace, offset)
		if p != nil {
			p.Stop(msg)
		}
		outputType, _ := cmd.Flags().GetString("output")

		if len(templates.Results) == 0 && utils.IsOutputPiped() {
			utils.OutputData([]interface{}{}, outputType)
			return
		}
		utils.OutputData(templates.Results, outputType)
	},
}

func init() {
	templateListCmd.PersistentFlags().IntP("limit", "l", 20, "Limit the number of templates to list")
	templateListCmd.PersistentFlags().Int("offset", 0, "Offset the number of templates to list (default: 0)")
	templateListCmd.PersistentFlags().StringP("mode", "m", "live", "Mode of templates to list (draft, live), default: live")
	templateListCmd.PersistentFlags().StringP("output", "o", "pretty", "Output Style (pretty, yaml, json)")
	templateListCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		cmd.Parent().HelpFunc()(cmd, args)
	})
	TemplateCmd.PersistentFlags().StringP("workspace", "w", "staging", "Workspace to list templates from")
	TemplateCmd.PersistentFlags().StringP("service-token", "s", "", "Service token (default: $SUPRSEND_SERVICE_TOKEN)")
	TemplateCmd.AddCommand(templateListCmd)
}
