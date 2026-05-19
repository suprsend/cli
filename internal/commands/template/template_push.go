/*
Copyright © 2025 SuprSend
*/
package template

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/cli/mgmnt"
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


func PushTemplate(mgmntClient *mgmnt.SS_MgmntClient, workspace, slug, templateDir, commitMessage string, commit bool, force bool, dryRun bool) error {
	templateData, err := readTemplateJSON(templateDir)
	if err != nil {
		return clierr.Wrap(err, clierr.CodeFileParseFailed, fmt.Sprintf("failed to read template.json for template %s", slug))
	}

	variants, err := readTemplateVariants(templateDir)
	if err != nil {
		return clierr.Wrap(err, clierr.CodeFileParseFailed, fmt.Sprintf("failed to read template %s", slug))
	}

	if dryRun {
		channels := make([]string, 0, len(variants))
		seen := map[string]bool{}
		for _, v := range variants {
			ch, _ := v["channel"].(string)
			if ch != "" && !seen[ch] {
				channels = append(channels, ch)
				seen[ch] = true
			}
		}
		log.Infof("DRY RUN: would push template '%s' — %d variant(s), channels: %v", slug, len(variants), channels)
		return nil
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
		return clierr.Wrap(err, clierr.CodeAPIInternal, fmt.Sprintf("failed to create template %s", slug))
	}

	var pushErrs []string
	for _, variant := range variants {
		if err := mgmntClient.PushTemplateVariant(workspace, slug, variant); err != nil {
			log.WithError(err).Errorf("templates/%s: failed to push variant", slug)
			pushErrs = append(pushErrs, err.Error())
		}
	}
	if len(pushErrs) > 0 {
		return clierr.New(fmt.Sprintf("templates/%s: failed to push %d variant(s): %s", slug, len(pushErrs), strings.Join(pushErrs, "; ")), clierr.CodeAPIInternal)
	}

	// Push mock_data.json if it exists
	mockDataFile := filepath.Join(templateDir, "mock_data.json")
	if mockDataBytes, err := os.ReadFile(mockDataFile); err == nil {
		var mockData map[string]any
		if err := json.Unmarshal(mockDataBytes, &mockData); err != nil {
			return clierr.Wrap(err, clierr.CodeFileParseFailed, fmt.Sprintf("failed to parse mock_data.json for template %s", slug))
		}
		if err := mgmntClient.PatchTemplateMockData(workspace, slug, mockData); err != nil {
			return clierr.Wrap(err, clierr.CodeAPIInternal, fmt.Sprintf("failed to push mock data for template %s", slug))
		}
	}

	// Push variant order if present in template.json
	if variantOrderRaw, ok := templateData["variant_order"]; ok && variantOrderRaw != nil {
		flatOrder, ok := variantOrderRaw.(map[string]any)
		if !ok {
			return clierr.New(fmt.Sprintf("invalid variant_order format for template %s", slug), clierr.CodeFileParseFailed)
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
			return clierr.Wrap(err, clierr.CodeAPIInternal, fmt.Sprintf("failed to push variant order for template %s", slug))
		}
	}

	if commit {
		if force {
			validateResp, err := mgmntClient.PreCommitValidate(workspace, slug)
			if err != nil {
				return clierr.Wrap(err, clierr.CodeAPIInternal, fmt.Sprintf("failed to pre-commit validate template %s", slug))
			}

			var validVariants []map[string]any
			for _, v := range validateResp.Variants {
				if len(v.Errors) > 0 {
					log.Warnf("Skipping variant %s/%s for template %s due to errors: %v", v.Channel, v.ID, slug, v.Errors)
					continue
				}
				validVariants = append(validVariants, map[string]any{
					"channel": v.Channel,
					"id":      v.ID,
				})
			}

			if len(validVariants) == 0 {
				log.Warnf("No valid variants to commit for template %s", slug)
				return nil
			}

			if err := mgmntClient.CommitTemplate(workspace, slug, commitMessage, validVariants); err != nil {
				return clierr.Wrap(err, clierr.CodeAPIInternal, fmt.Sprintf("failed to commit template %s", slug))
			}
		} else {
			if err := mgmntClient.CommitTemplate(workspace, slug, commitMessage, nil); err != nil {
				return clierr.Wrap(err, clierr.CodeAPIInternal, fmt.Sprintf("failed to commit template %s", slug))
			}
		}
	}

	return nil
}

var templatePushCmd = &cobra.Command{
	Use:   "push [<slug>]",
	Short: "Push templates and their variants from local to SuprSend workspace",
	Long:  `Push templates and their variants from local to SuprSend workspace. Pass a slug as a positional argument or via --slug to push a single template, or omit to push all.`,
	Example: `  # Push all templates from default directory
  suprsend template push

  # Push a single template and commit to live immediately
  suprsend template push welcome-email --commit

  # Dry run: preview what would be pushed without making changes
  suprsend template push --dry-run`,
	Annotations: map[string]string{
		"skills:tip.a-draft":  "Push writes to the **draft** state. Run `suprsend template commit` to promote draft → live.",
		"skills:tip.b-dryrun": "Pair with `--dry-run` to validate the template server-side without writing to the draft. Pair with `--commit` to push + commit in one step.",
	},
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		path, _ := cmd.Flags().GetString("dir")
		commit, _ := cmd.Flags().GetBool("commit")
		commitMessage, _ := cmd.Flags().GetString("commit-message")
		slug := utils.ResolveSlug(cmd, args)
		force, _ := cmd.Flags().GetBool("force")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		var dryRunSlugs []string

		if path == "" {
			path = filepath.Join(".", "suprsend", "templates")
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			log.Errorf("Directory %s does not exist", path)
			return clierr.Wrap(err, clierr.CodeFileNotFound, "")
		}

		mgmntClient := utils.GetSuprSendMgmntClient()

		hasError := false
		var spinner *utils.Spinner

		if slug != "" {
			templateDir := filepath.Join(path, slug)
			if _, err := os.Stat(templateDir); os.IsNotExist(err) {
				return clierr.Wrap(err, clierr.CodeFileNotFound, fmt.Sprintf("template directory %s does not exist", templateDir))
			}

			spinner = utils.NewSpinner(fmt.Sprintf("Pushing template %s...", slug))
			err := PushTemplate(mgmntClient, workspace, slug, templateDir, commitMessage, commit, force, dryRun)
			if err != nil {
				spinner.Stop("")
				return err
			}
			if dryRun {
				spinner.Stop(fmt.Sprintf("(dry run) %s", slug))
			} else {
				spinner.Stop(fmt.Sprintf("Pushed template: %s", slug))
			}
			return nil
		}

		stats := &TemplatePushStats{
			Errors: []string{},
		}

		// Push all templates
		entries, err2 := os.ReadDir(path)
		if err2 != nil {
			log.WithError(err2).Errorf("Failed to read templates directory")
			return clierr.Wrap(err2, clierr.CodeFileNotFound, "")
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

			if !hasError {
				spinner = utils.NewSpinner(fmt.Sprintf("Pushing template %s...", templateSlug))
			}

			if err := PushTemplate(mgmntClient, workspace, templateSlug, templateDir, commitMessage, commit, force, dryRun); err != nil {
				spinner.Stop("")
				hasError = true
				log.WithError(err).Errorf("templates/%s: failed to push", templateSlug)
				stats.Failed++
				stats.Errors = append(stats.Errors, fmt.Sprintf("templates/%s: failed to push: %v", templateSlug, err))
				continue
			}

			stats.Success++
			if dryRun {
				dryRunSlugs = append(dryRunSlugs, templateSlug)
				spinner.Stop(fmt.Sprintf("(dry run) %s", templateSlug))
			} else {
				spinner.Stop(fmt.Sprintf("Pushed template: %s", templateSlug))
			}
			hasError = false
		}

		if dryRun {
			action := "push"
			if commit {
				action = "push and commit"
			}
			log.Infof("DRY RUN: would %s %d template(s) to %s", action, len(dryRunSlugs), workspace)
			for _, s := range dryRunSlugs {
				log.Infof("  - %s", s)
			}
			return nil
		}

		log.Info("=== Template Push Summary ===")
		log.Infof("Total templates processed: %d", stats.Total)
		log.Infof("Successfully pushed: %d", stats.Success)
		log.Infof("Failed to push: %d", stats.Failed)

		if stats.Failed > 0 {
			log.Info("Failed templates:")
			for _, errorMsg := range stats.Errors {
				log.Infof("  - %s", errorMsg)
			}
			return clierr.New(fmt.Sprintf("%d template(s) failed to push", stats.Failed), clierr.CodeAPIInternal)
		}
		return nil
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
	templatePushCmd.PersistentFlags().BoolP("dry-run", "n", false, "Print what would be pushed without making any changes")
	TemplateCmd.AddCommand(templatePushCmd)
}
