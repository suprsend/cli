/*
Copyright © 2025 SuprSend
*/
package template

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/utils"
)

var templateGetCmd = &cobra.Command{
	Use:   "get [<slug>]",
	Short: "Get template details including variants",
	Long:  "Retrieve a specific template by slug, including all its channel variants and mock data. Use --mode to switch between draft and live versions.",
	Example: `  # Get a template by slug (positional)
  suprsend template get welcome-email

  # Get using the flag form
  suprsend template get --slug welcome-email

  # Get the draft version
  suprsend template get welcome-email --mode draft`,
	Args:  cobra.MaximumNArgs(1),
	Annotations: map[string]string{
		"skills:tip:output": "Use `-o json` for machine-readable JSON output, `-o yaml` for YAML. Default `-o json` outputs the full template with variants.",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		slug := utils.ResolveSlug(cmd, args)
		if slug == "" {
			return clierr.New("slug is required: provide it as a positional argument or via --slug", clierr.CodeInvalidUsage)
		}
		workspace, _ := cmd.Flags().GetString("workspace")
		mode, _ := cmd.Flags().GetString("mode")
		outputType, _ := cmd.Flags().GetString("output")
		if err := utils.ValidateOutputType(outputType, "json", "yaml", "pretty"); err != nil {
			return err
		}

		mgmntClient := utils.GetSuprSendMgmntClient()

		spinner := utils.NewSpinner("Getting template...")

		template, err := mgmntClient.GetTemplate(workspace, slug, mode)
		if err != nil {
			spinner.Stop("")
			log.WithError(err).Errorf("Error getting template")
			return clierr.Wrap(err, clierr.CodeAPIInternal, "")
		}

		variants, err := mgmntClient.GetTemplateVariants(workspace, slug, mode)
		if err != nil {
			spinner.Stop("")
			log.WithError(err).Errorf("Error getting template variants")
			return clierr.Wrap(err, clierr.CodeAPIInternal, "")
		}

		mockData, err := mgmntClient.GetTemplateMockData(workspace, slug)
		if err != nil {
			log.WithError(err).Warnf("Couldn't fetch mock data for template: %s", slug)
		}

		spinner.Stop(fmt.Sprintf("Successfully got template '%s' with %d variant(s)", slug, len(variants)))

		result := TemplateResult{
			Slug:            template.Slug,
			Name:            template.Name,
			EnabledChannels: template.EnabledChannels,
			Variants:        variants,
			MockData:        mockData,
		}
		utils.OutputData(result, outputType)
		return nil
	},
}

func init() {
	templateGetCmd.PersistentFlags().StringP("slug", "g", "", "Template slug")
	templateGetCmd.PersistentFlags().StringP("mode", "m", "live", "Version mode: draft or live")
	templateGetCmd.PersistentFlags().StringP("output", "o", "json", "Output format: json, yaml, or pretty")
	TemplateCmd.AddCommand(templateGetCmd)
}
