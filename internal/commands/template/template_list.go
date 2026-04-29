/*
Copyright © 2025 SuprSend
*/
package template

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/utils"
)

var templateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List templates for a workspace",
	Long:  `List templates in a workspace with pagination. Returns template slug, name, and enabled channel info. Use --mode to switch between draft and live versions.`,
	Example: `  # List all templates (live mode)
  suprsend template list

  # List draft templates
  suprsend template list --mode draft

  # Paginate with JSON output
  suprsend template list --limit 50 --output json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		spinner := utils.NewSpinner("Loading...")
		workspace, _ := cmd.Flags().GetString("workspace")
		mgmntClient := utils.GetSuprSendMgmntClient()

		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")
		mode, _ := cmd.Flags().GetString("mode")

		templates, err := mgmntClient.ListTemplates(workspace, limit, offset, mode)
		if err != nil {
			return err
		}

		spinner.Stop(fmt.Sprintf("Listed %d templates from %s with offset %d", len(templates.Results), workspace, offset))
		outputType, _ := cmd.Flags().GetString("output")
		if err := utils.ValidateOutputType(outputType, "pretty", "json", "yaml"); err != nil {
			return err
		}

		if len(templates.Results) == 0 && utils.IsOutputPiped() {
			utils.OutputData([]interface{}{}, outputType)
			return nil
		}
		utils.OutputData(templates.Results, outputType)
		return nil
	},
}

func init() {
	templateListCmd.PersistentFlags().IntP("limit", "l", 20, "Limit the number of templates to list")
	templateListCmd.PersistentFlags().Int("offset", 0, "Offset the number of templates to list (default: 0)")
	templateListCmd.PersistentFlags().StringP("mode", "m", "live", "Version mode: draft or live")
	templateListCmd.PersistentFlags().StringP("output", "o", "pretty", "Output Style (pretty, yaml, json)")
	templateListCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		cmd.Parent().HelpFunc()(cmd, args)
	})
	TemplateCmd.PersistentFlags().StringP("workspace", "w", "staging", "Workspace to list templates from")
	TemplateCmd.PersistentFlags().StringP("service-token", "s", "", "Service token (default: $SUPRSEND_SERVICE_TOKEN)")
	TemplateCmd.AddCommand(templateListCmd)
}
