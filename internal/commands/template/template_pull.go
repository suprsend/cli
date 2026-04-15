/*
Copyright © 2025 SuprSend
*/
package template

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/cli/mgmnt"
	"github.com/yarlson/pin"
)

type templateResult struct {
	Slug            string                      `json:"slug"`
	Name            string                      `json:"name"`
	EnabledChannels []string                    `json:"enabled_channels"`
	Variants        []map[string]any            `json:"variants"`
	MockData        map[string]any              `json:"mock_data,omitempty"`
	VariantOrder    *mgmnt.VariantOrderResponse `json:"variant_order,omitempty"`
}

var templatePullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull templates and their variants from SuprSend workspace",
	Long:  `Pull templates and their variants from SuprSend workspace`,
	Run: func(cmd *cobra.Command, args []string) {
		workspace, _ := cmd.Flags().GetString("workspace")
		mode, _ := cmd.Flags().GetString("mode")
		slug, _ := cmd.Flags().GetString("slug")
		outputDir, _ := cmd.Flags().GetString("dir")
		force, _ := cmd.Flags().GetBool("force")

		// Directory resolution
		if outputDir == "" {
			outputDir = filepath.Join(".", "suprsend", "templates")
			if _, err := os.Stat(outputDir); os.IsNotExist(err) {
				if force {
					fmt.Fprintf(os.Stdout, "Using default directory: %s\n", outputDir)
				} else {
					outputDir = promptForOutputDirectory()
				}
			}
			if outputDir == "" {
				fmt.Fprintf(os.Stdout, "No output directory specified. Exiting.\n")
				return
			}
		}

		if err := ensureOutputDirectory(outputDir); err != nil {
			fmt.Fprintf(os.Stdout, "Error with output directory: %v\n", err)
			return
		}

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

		var results []templateResult

		if slug != "" {
			tmpl, err := mgmntClient.GetTemplate(workspace, slug, mode)
			if err != nil {
				if p != nil {
					p.Stop("Failed")
				}
				log.WithError(err).Error("Couldn't fetch template")
				return
			}
			variants, err := mgmntClient.GetTemplateVariants(workspace, slug, mode)
			if err != nil {
				if p != nil {
					p.Stop("Failed")
				}
				log.WithError(err).Error("Couldn't fetch template variants")
				return
			}
			mockData, err := mgmntClient.GetTemplateMockData(workspace, slug)
			if err != nil {
				log.WithError(err).Warnf("Couldn't fetch mock data for template: %s", slug)
			}
			variantOrder, err := mgmntClient.GetVariantOrder(workspace, slug, mode)
			if err != nil {
				log.WithError(err).Warnf("Couldn't fetch variant order for template: %s", slug)
			}
			results = append(results, templateResult{Slug: slug, Name: tmpl.Name, EnabledChannels: tmpl.EnabledChannels, Variants: variants, MockData: mockData, VariantOrder: variantOrder})
		} else {
			// Fetch all template slugs
			templates, err := mgmntClient.ListTemplates(workspace, math.MaxInt32, 0, mode)
			if err != nil {
				if p != nil {
					p.Stop("Failed")
				}
				log.WithError(err).Error("Couldn't fetch templates")
				return
			}

			for _, t := range templates.Results {
				variants, err := mgmntClient.GetTemplateVariants(workspace, t.Slug, mode)
				if err != nil {
					log.WithError(err).Errorf("Couldn't fetch variants for template: %s", t.Slug)
					continue
				}
				mockData, err := mgmntClient.GetTemplateMockData(workspace, t.Slug)
				if err != nil {
					log.WithError(err).Warnf("Couldn't fetch mock data for template: %s", t.Slug)
				}
				variantOrder, err := mgmntClient.GetVariantOrder(workspace, t.Slug, mode)
				if err != nil {
					log.WithError(err).Warnf("Couldn't fetch variant order for template: %s", t.Slug)
				}
				results = append(results, templateResult{
					Slug:            t.Slug,
					Name:            t.Name,
					EnabledChannels: t.EnabledChannels,
					Variants:        variants,
					MockData:        mockData,
					VariantOrder:    variantOrder,
				})
			}
		}

		totalVariants := 0
		for _, r := range results {
			totalVariants += len(r.Variants)
		}

		msg := fmt.Sprintf("Pulled %d templates with %d variants from %s", len(results), totalVariants, workspace)
		if p != nil {
			p.Stop(msg)
		}

		stats, err := WriteTemplatesToFiles(results, outputDir)
		if err != nil {
			fmt.Fprintf(os.Stdout, "Error: Failed to save templates: %v\n", err)
			return
		}

		fmt.Fprintf(os.Stdout, "\nPull Summary: %d total, %d success, %d failed\n", stats.Total, stats.Success, stats.Failed)
		if len(stats.Errors) > 0 {
			fmt.Fprintf(os.Stdout, "Errors:\n")
			for _, e := range stats.Errors {
				fmt.Fprintf(os.Stdout, "  - %s\n", e)
			}
		}
	},
}

func init() {
	templatePullCmd.PersistentFlags().StringP("mode", "m", "live", "Mode of templates to pull (draft, live)")
	templatePullCmd.PersistentFlags().StringP("slug", "g", "", "Slug of a specific template to pull")
	templatePullCmd.PersistentFlags().StringP("dir", "d", "", "Output directory for templates (default: ./suprsend/templates)")
	templatePullCmd.PersistentFlags().BoolP("force", "f", false, "Force using default directory without prompting")
	TemplateCmd.AddCommand(templatePullCmd)
}
