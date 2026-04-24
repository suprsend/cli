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

type TemplateResult struct {
	Slug            string                      `json:"slug"`
	Name            string                      `json:"name"`
	EnabledChannels []string                    `json:"enabled_channels"`
	Variants        []map[string]any            `json:"variants"`
	MockData        map[string]any              `json:"mock_data,omitempty"`
	VariantOrder    *mgmnt.VariantOrderResponse `json:"variant_order,omitempty"`
}

// FetchTemplates fetches either one template (when slug != "") or all templates
// from the workspace, returning the assembled results ready to hand to
// WriteTemplatesToFiles. Per-template fetch errors when slug == "" are logged
// and the offending template is skipped; only failures while listing templates
// or fetching a specifically requested slug are returned.
func FetchTemplates(client *mgmnt.SS_MgmntClient, workspace, mode, slug string) ([]TemplateResult, error) {
	var results []TemplateResult

	if slug != "" {
		tmpl, err := client.GetTemplate(workspace, slug, mode)
		if err != nil {
			return nil, fmt.Errorf("couldn't fetch template %s: %w", slug, err)
		}
		variants, err := client.GetTemplateVariants(workspace, slug, mode)
		if err != nil {
			return nil, fmt.Errorf("couldn't fetch variants for template %s: %w", slug, err)
		}
		mockData, err := client.GetTemplateMockData(workspace, slug)
		if err != nil {
			log.WithError(err).Warnf("Couldn't fetch mock data for template: %s", slug)
		}
		variantOrder, err := client.GetVariantOrder(workspace, slug, mode)
		if err != nil {
			log.WithError(err).Warnf("Couldn't fetch variant order for template: %s", slug)
		}
		results = append(results, TemplateResult{
			Slug:            slug,
			Name:            tmpl.Name,
			EnabledChannels: tmpl.EnabledChannels,
			Variants:        variants,
			MockData:        mockData,
			VariantOrder:    variantOrder,
		})
		return results, nil
	}

	templates, err := client.ListTemplates(workspace, math.MaxInt32, 0, mode)
	if err != nil {
		return nil, fmt.Errorf("couldn't fetch templates: %w", err)
	}

	for _, t := range templates.Results {
		variants, err := client.GetTemplateVariants(workspace, t.Slug, mode)
		if err != nil {
			log.WithError(err).Errorf("Couldn't fetch variants for template: %s", t.Slug)
			continue
		}
		mockData, err := client.GetTemplateMockData(workspace, t.Slug)
		if err != nil {
			log.WithError(err).Warnf("Couldn't fetch mock data for template: %s", t.Slug)
		}
		variantOrder, err := client.GetVariantOrder(workspace, t.Slug, mode)
		if err != nil {
			log.WithError(err).Warnf("Couldn't fetch variant order for template: %s", t.Slug)
		}
		results = append(results, TemplateResult{
			Slug:            t.Slug,
			Name:            t.Name,
			EnabledChannels: t.EnabledChannels,
			Variants:        variants,
			MockData:        mockData,
			VariantOrder:    variantOrder,
		})
	}
	return results, nil
}

var templatePullCmd = &cobra.Command{
	Use:   "pull [<slug>]",
	Short: "Pull templates and their variants from SuprSend workspace",
	Long:  `Pull templates and their variants from SuprSend workspace. Pass a slug as a positional argument or via --slug to pull a single template, or omit to pull all.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		workspace, _ := cmd.Flags().GetString("workspace")
		mode, _ := cmd.Flags().GetString("mode")
		slug := utils.ResolveSlug(cmd, args)
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

		results, err := FetchTemplates(mgmntClient, workspace, mode, slug)
		if err != nil {
			if p != nil {
				p.Stop("Failed")
			}
			log.WithError(err).Error(err.Error())
			return
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
	templatePullCmd.PersistentFlags().BoolP("force", "F", false, "Force using default directory without prompting")
	TemplateCmd.AddCommand(templatePullCmd)
}
