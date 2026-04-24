/*
Copyright © 2025 SuprSend
*/
package template

import (
	"fmt"
	"math"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/cli/mgmnt"
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
					log.Infof("Using default directory: %s", outputDir)
				} else {
					od, success := promptForOutputDirectory()
					if !success {
						return
					}
					outputDir = od
				}
			}
			if outputDir == "" {
				log.Info("No output directory specified. Exiting.")
				return
			}
		}

		if err := ensureOutputDirectory(outputDir); err != nil {
			log.Errorf("Error with output directory: %v", err)
			return
		}

		spinner := utils.NewSpinner("Loading...")

		mgmntClient := utils.GetSuprSendMgmntClient()

		results, err := FetchTemplates(mgmntClient, workspace, mode, slug)
		if err != nil {
			spinner.Stop("Failed")
			log.WithError(err).Error(err.Error())
			return
		}

		totalVariants := 0
		for _, r := range results {
			totalVariants += len(r.Variants)
		}

		spinner.Stop(fmt.Sprintf("Pulled %d templates with %d variants from %s", len(results), totalVariants, workspace))

		stats, err := WriteTemplatesToFiles(results, outputDir)
		if err != nil {
			log.Errorf("Failed to save templates: %v", err)
			return
		}

		log.Infof("Pull Summary: %d total, %d success, %d failed", stats.Total, stats.Success, stats.Failed)
		if len(stats.Errors) > 0 {
			log.Info("Errors:")
			for _, e := range stats.Errors {
				log.Infof("  - %s", e)
			}
		}
	},
}

func init() {
	templatePullCmd.PersistentFlags().StringP("mode", "m", "live", "Version mode: draft or live")
	templatePullCmd.PersistentFlags().StringP("slug", "g", "", "Slug of a specific template to pull")
	templatePullCmd.PersistentFlags().StringP("dir", "d", "", "Output directory for templates (default: ./suprsend/templates)")
	templatePullCmd.PersistentFlags().BoolP("force", "F", false, "Force using default directory without prompting")
	TemplateCmd.AddCommand(templatePullCmd)
}
