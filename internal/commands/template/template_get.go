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

var templateGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get template details including variants",
	Long:  "Retrieve a specific template by slug, including all its channel variants and mock data. Requires --slug. Use --mode to switch between draft and live versions.",
	Annotations: map[string]string{
		"skills:tip:output": "Use `-o json` for machine-readable JSON output, `-o yaml` for YAML. Default `-o json` outputs the full template with variants.",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		slug, _ := cmd.Flags().GetString("slug")
		if slug == "" {
			log.Error("Template slug is required. Example: suprsend template get --slug <slug>")
			return fmt.Errorf("template slug is required. Example: suprsend template get --slug <slug>")
		}
		workspace, _ := cmd.Flags().GetString("workspace")
		mode, _ := cmd.Flags().GetString("mode")
		outputType, _ := cmd.Flags().GetString("output")

		mgmntClient := utils.GetSuprSendMgmntClient()

		var p *pin.Pin
		var cancel context.CancelFunc
		if !utils.IsOutputPiped() {
			p = pin.New("Getting template...",
				pin.WithSpinnerColor(pin.ColorCyan),
				pin.WithTextColor(pin.ColorYellow),
			)
			cancel = p.Start(context.Background())
		}

		template, err := mgmntClient.GetTemplate(workspace, slug, mode)
		if err != nil {
			if p != nil {
				p.Stop("")
				cancel()
			}
			log.WithError(err).Errorf("Error getting template")
			return err
		}

		variants, err := mgmntClient.GetTemplateVariants(workspace, slug, mode)
		if err != nil {
			if p != nil {
				p.Stop("")
				cancel()
			}
			log.WithError(err).Errorf("Error getting template variants")
			return err
		}

		mockData, err := mgmntClient.GetTemplateMockData(workspace, slug)
		if err != nil {
			log.WithError(err).Warnf("Couldn't fetch mock data for template: %s", slug)
		}

		if p != nil {
			p.Stop(fmt.Sprintf("Successfully got template '%s' with %d variant(s)", slug, len(variants)))
			cancel()
		}

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
	templateGetCmd.PersistentFlags().StringP("slug", "g", "", "Template slug to retrieve (required)")
	templateGetCmd.PersistentFlags().StringP("mode", "m", "live", "Version mode: draft or live")
	templateGetCmd.PersistentFlags().StringP("output", "o", "json", "Output format: json, yaml, or pretty")
	TemplateCmd.AddCommand(templateGetCmd)
}
