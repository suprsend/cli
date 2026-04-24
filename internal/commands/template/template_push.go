/*
Copyright © 2025 SuprSend
*/
package template

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/cli/mgmnt"
	"github.com/yarlson/pin"
)

func readTemplateJSON(templateDir string) (map[string]any, error) {
	templateFile := filepath.Join(templateDir, "template.json")
	data, err := os.ReadFile(templateFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read template.json: %w", err)
	}
	var templateData map[string]any
	if err := json.Unmarshal(data, &templateData); err != nil {
		return nil, fmt.Errorf("failed to parse template.json: %w", err)
	}
	return templateData, nil
}


func PushTemplate(mgmntClient *mgmnt.SS_MgmntClient, workspace, slug, templateDir, commitMessage string, commit bool, force bool, stats *TemplatePushStats) {
	templateData, err := readTemplateJSON(templateDir)
	if err != nil {
		log.WithError(err).Errorf("Failed to read template.json for template %s", slug)
		stats.Failed++
		stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to read template.json for template %s: %v", slug, err))
		return
	}

	variants, err := readTemplateVariants(templateDir)
	if err != nil {
		log.WithError(err).Errorf("Failed to read template %s", slug)
		stats.Failed++
		stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to read template %s: %v", slug, err))
		return
	}

	var enabledChannels []string
	if channels, ok := templateData["enabled_channels"].([]any); ok {
		for _, ch := range channels {
			if s, ok := ch.(string); ok {
				enabledChannels = append(enabledChannels, s)
			}
		}
	}
	if err := mgmntClient.CreateTemplate(workspace, slug, enabledChannels); err != nil {
		log.WithError(err).Errorf("Failed to create template %s", slug)
		stats.Failed++
		stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to create template %s: %v", slug, err))
		return
	}

	var pushFailed bool
	for _, variant := range variants {
		if err := mgmntClient.PushTemplateVariant(workspace, slug, variant); err != nil {
			log.WithError(err).Errorf("Failed to push variant for template %s", slug)
			stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to push variant for template %s: %v", slug, err))
			pushFailed = true
		}
	}

	if pushFailed {
		stats.Failed++
		return
	}

	// Push mock_data.json if it exists
	mockDataFile := filepath.Join(templateDir, "mock_data.json")
	if mockDataBytes, err := os.ReadFile(mockDataFile); err == nil {
		var mockData map[string]any
		if err := json.Unmarshal(mockDataBytes, &mockData); err != nil {
			log.WithError(err).Errorf("Failed to parse mock_data.json for template %s", slug)
			stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to parse mock_data.json for template %s: %v", slug, err))
			stats.Failed++
			return
		}
		if err := mgmntClient.PatchTemplateMockData(workspace, slug, mockData); err != nil {
			log.WithError(err).Errorf("Failed to push mock data for template %s", slug)
			stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to push mock data for template %s: %v", slug, err))
			stats.Failed++
			return
		}
	}

	// Push variant order if present in template.json
	if variantOrderRaw, ok := templateData["variant_order"]; ok && variantOrderRaw != nil {
		flatOrder, ok := variantOrderRaw.(map[string]any)
		if !ok {
			log.Errorf("Invalid variant_order format for template %s", slug)
			stats.Errors = append(stats.Errors, fmt.Sprintf("Invalid variant_order format for template %s", slug))
			stats.Failed++
			return
		}
		channelMap := map[string]*mgmnt.VariantOrderChannel{}
		for key, variantsRaw := range flatOrder {
			parts := strings.SplitN(key, "/", 2)
			channel := parts[0]
			var tenantIDPtr *string
			if len(parts) == 2 {
				t := parts[1]
				tenantIDPtr = &t
			}
			var variants []string
			if variantsList, ok := variantsRaw.([]any); ok {
				for _, v := range variantsList {
					if s, ok := v.(string); ok {
						variants = append(variants, s)
					}
				}
			}
			if _, exists := channelMap[channel]; !exists {
				channelMap[channel] = &mgmnt.VariantOrderChannel{Channel: channel}
			}
			channelMap[channel].Tenants = append(channelMap[channel].Tenants, mgmnt.VariantOrderTenant{
				TenantID: tenantIDPtr,
				Variants: variants,
			})
		}
		var variantOrder mgmnt.VariantOrderResponse
		for _, ch := range channelMap {
			variantOrder.Channels = append(variantOrder.Channels, *ch)
		}
		if err := mgmntClient.PostVariantOrder(workspace, slug, "draft", &variantOrder); err != nil {
			log.WithError(err).Errorf("Failed to push variant order for template %s", slug)
			stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to push variant order for template %s: %v", slug, err))
			stats.Failed++
			return
		}
	}

	if commit {
		if force {
			validateResp, err := mgmntClient.PreCommitValidate(workspace, slug)
			if err != nil {
				log.WithError(err).Errorf("Failed to pre-commit validate template %s", slug)
				stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to pre-commit validate template %s: %v", slug, err))
				stats.Failed++
				return
			}

			var validVariants []map[string]any
			for _, v := range validateResp.Variants {
				if len(v.Errors) > 0 {
					log.Warnf("Skipping variant %s/%s for template %s due to errors: %v", v.Channel, v.ID, slug, v.Errors)
					stats.Errors = append(stats.Errors, fmt.Sprintf("Skipped variant %s/%s for template %s (has errors)", v.Channel, v.ID, slug))
					continue
				}
				validVariants = append(validVariants, map[string]any{
					"channel": v.Channel,
					"id":      v.ID,
				})
			}

			if len(validVariants) == 0 {
				log.Warnf("No valid variants to commit for template %s", slug)
				return
			}

			if err := mgmntClient.CommitTemplate(workspace, slug, commitMessage, validVariants); err != nil {
				log.WithError(err).Errorf("Failed to commit template %s", slug)
				stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to commit template %s: %v", slug, err))
				stats.Failed++
				return
			}
		} else {
			if err := mgmntClient.CommitTemplate(workspace, slug, commitMessage, nil); err != nil {
				log.WithError(err).Errorf("Failed to commit template %s", slug)
				stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to commit template %s: %v", slug, err))
				stats.Failed++
				return
			}
		}
	}

	stats.Success++
}

var templatePushCmd = &cobra.Command{
	Use:   "push [<slug>]",
	Short: "Push templates and their variants from local to SuprSend workspace",
	Long:  `Push templates and their variants from local to SuprSend workspace. Pass a slug as a positional argument or via --slug to push a single template, or omit to push all.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		workspace, _ := cmd.Flags().GetString("workspace")
		path, _ := cmd.Flags().GetString("dir")
		commit, _ := cmd.Flags().GetBool("commit")
		commitMessage, _ := cmd.Flags().GetString("commit-message")
		slug := utils.ResolveSlug(cmd, args)
		force, _ := cmd.Flags().GetBool("force")

		if path == "" {
			path = filepath.Join(".", "suprsend", "templates")
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			log.Errorf("Directory %s does not exist", path)
			return
		}

		mgmntClient := utils.GetSuprSendMgmntClient()

		stats := &TemplatePushStats{
			Errors: []string{},
		}

		hasError := false
		var p *pin.Pin
		var cancel context.CancelFunc

		if slug != "" {
			// Push a single template by slug
			stats.Total = 1
			templateDir := filepath.Join(path, slug)

			if _, err := os.Stat(templateDir); os.IsNotExist(err) {
				log.Errorf("Template directory %s does not exist", templateDir)
				stats.Failed++
				stats.Errors = append(stats.Errors, fmt.Sprintf("Template directory %s does not exist", templateDir))
			} else {
				if !utils.IsOutputPiped() {
					p = pin.New(fmt.Sprintf("Pushing template %s...", slug),
						pin.WithSpinnerColor(pin.ColorCyan),
						pin.WithTextColor(pin.ColorYellow),
					)
					cancel = p.Start(context.Background())
				}

				PushTemplate(mgmntClient, workspace, slug, templateDir, commitMessage, commit, force, stats)

				if p != nil && cancel != nil {
					if stats.Success > 0 {
						p.Stop(fmt.Sprintf("Pushed template: %s", slug))
					} else {
						p.Stop("")
					}
					cancel()
				} else if stats.Success > 0 {
					fmt.Fprintf(os.Stdout, "Pushed template: %s\n", slug)
				}
			}
		} else {
			// Push all templates
			entries, err := os.ReadDir(path)
			if err != nil {
				log.WithError(err).Errorf("Failed to read templates directory")
				return
			}

			for _, entry := range entries {
				if entry.IsDir() {
					stats.Total++
				}
			}

			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}

				templateSlug := entry.Name()
				templateDir := filepath.Join(path, templateSlug)

				if !hasError && !utils.IsOutputPiped() {
					p = pin.New(fmt.Sprintf("Pushing template %s...", templateSlug),
						pin.WithSpinnerColor(pin.ColorCyan),
						pin.WithTextColor(pin.ColorYellow),
					)
					cancel = p.Start(context.Background())
				}

				prevFailed := stats.Failed
				PushTemplate(mgmntClient, workspace, templateSlug, templateDir, commitMessage, commit, force, stats)

				if stats.Failed > prevFailed {
					if p != nil && cancel != nil {
						p.Stop("")
						cancel()
						p = nil
						cancel = nil
					}
					hasError = true
					continue
				}

				if p != nil && cancel != nil {
					p.Stop(fmt.Sprintf("Pushed template: %s", templateSlug))
					cancel()
					p = nil
					cancel = nil
				} else {
					fmt.Fprintf(os.Stdout, "Pushed template: %s\n", templateSlug)
				}
				hasError = false
			}
		}

		fmt.Fprintf(os.Stdout, "\n=== Template Push Summary ===\n")
		fmt.Fprintf(os.Stdout, "Total templates processed: %d\n", stats.Total)
		fmt.Fprintf(os.Stdout, "Successfully pushed: %d\n", stats.Success)
		fmt.Fprintf(os.Stdout, "Failed to push: %d\n", stats.Failed)

		if stats.Failed > 0 {
			fmt.Fprintf(os.Stdout, "\nFailed templates:\n")
			for _, errorMsg := range stats.Errors {
				fmt.Fprintf(os.Stdout, "  - %s\n", errorMsg)
			}
		}
	},
}

// readTemplateVariants walks a template directory and returns all reassembled variants.
func readTemplateVariants(templateDir string) ([]map[string]any, error) {
	var variants []map[string]any

	err := filepath.WalkDir(templateDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "variant.json" {
			return nil
		}

		// Validate path structure: only accept known layouts
		// Valid: <channel>/<variant_name>/variant.json (depth 2)
		// Valid: <channel>/_tenants/<tenant>/<variant_name>/variant.json (depth 4)
		rel, _ := filepath.Rel(templateDir, filepath.Dir(path))
		parts := strings.Split(rel, string(filepath.Separator))
		if len(parts) == 2 {
			// ok: channel/variant_name
		} else if len(parts) == 4 && parts[1] == "_tenants" {
			// ok: channel/_tenants/tenant/variant_name
		} else {
			return fmt.Errorf("unexpected variant.json at %s: expected <channel>/<variant>/variant.json or <channel>/_tenants/<tenant>/<variant>/variant.json", rel)
		}

		variantDir := filepath.Dir(path)
		variant, err := readAndAssembleVariant(variantDir)
		if err != nil {
			return fmt.Errorf("failed to read variant at %s: %w", variantDir, err)
		}

		variants = append(variants, variant)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return variants, nil
}

func init() {
	templatePushCmd.PersistentFlags().StringP("dir", "d", "", "Input directory for templates (default: ./suprsend/templates)")
	templatePushCmd.PersistentFlags().BoolP("commit", "c", false, "Commit the pushed templates to live")
	templatePushCmd.PersistentFlags().String("commit-message", "", "Commit message describing the changes")
	templatePushCmd.PersistentFlags().StringP("slug", "g", "", "Slug of a specific template to push")
	templatePushCmd.PersistentFlags().BoolP("force", "F", false, "Force commit by skipping variants with errors")
	TemplateCmd.AddCommand(templatePushCmd)
}
