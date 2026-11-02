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

// PushTemplate pushes one template directory and, with commit, promotes it
// to live. It returns the report status for the template (pushed, committed,
// unchanged, or would-push/would-commit on a dry run) and a detail note.
// With skipInvalid, only the variants that pass pre-commit validation are
// committed; otherwise all of them are, and an invalid variant fails the commit.
func PushTemplate(mgmntClient *mgmnt.SS_MgmntClient, workspace, slug, templateDir, commitMessage string, commit, skipInvalid, dryRun bool) (string, string, error) {
	templateData, err := readTemplateJSON(templateDir)
	if err != nil {
		return "", "", clierr.Wrap(err, clierr.CodeFileParseFailed, fmt.Sprintf("failed to read template.json for template %s", slug))
	}

	variants, err := readTemplateVariants(templateDir)
	if err != nil {
		return "", "", clierr.Wrap(err, clierr.CodeFileParseFailed, fmt.Sprintf("failed to read template %s", slug))
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
		status := utils.StatusWouldPush
		if commit {
			status = utils.StatusWouldCommit
		}
		return status, fmt.Sprintf("%d variant(s), channels: %s", len(variants), strings.Join(channels, ", ")), nil
	}

	if err := mgmntClient.CreateTemplate(workspace, slug, templateFields(templateData)); err != nil {
		return "", "", clierr.Wrap(err, clierr.CodeAPIInternal, fmt.Sprintf("failed to create template %s", slug))
	}

	var pushErrs []string
	for _, variant := range variants {
		if err := mgmntClient.PushTemplateVariant(workspace, slug, variant); err != nil {
			pushErrs = append(pushErrs, err.Error())
		}
	}
	if len(pushErrs) > 0 {
		return "", "", clierr.New(fmt.Sprintf("templates/%s: failed to push %d variant(s): %s", slug, len(pushErrs), strings.Join(pushErrs, "; ")), clierr.CodeAPIInternal)
	}

	// Push mock_data.json if it exists
	mockDataFile := filepath.Join(templateDir, "mock_data.json")
	if mockDataBytes, err := os.ReadFile(mockDataFile); err == nil {
		var mockData map[string]any
		if err := json.Unmarshal(mockDataBytes, &mockData); err != nil {
			return "", "", clierr.Wrap(err, clierr.CodeFileParseFailed, fmt.Sprintf("failed to parse mock_data.json for template %s", slug))
		}
		stored, err := mgmntClient.PatchTemplateMockData(workspace, slug, mockData)
		if err != nil {
			return "", "", clierr.Wrap(err, clierr.CodeAPIInternal, fmt.Sprintf("failed to push mock data for template %s", slug))
		}
		warnDroppedMockData(slug, mockData, stored)
	}

	// Push variant order if present in template.json
	if variantOrderRaw, ok := templateData["variant_order"]; ok && variantOrderRaw != nil {
		flatOrder, ok := variantOrderRaw.(map[string]any)
		if !ok {
			return "", "", clierr.New(fmt.Sprintf("invalid variant_order format for template %s", slug), clierr.CodeFileParseFailed)
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
			variants := []string{} // the API rejects null
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
		// The API rejects null channels, so send [] for a template with no variants.
		variantOrder := mgmnt.VariantOrderResponse{Channels: []mgmnt.VariantOrderChannel{}}
		for _, ch := range channelMap {
			variantOrder.Channels = append(variantOrder.Channels, *ch)
		}
		if err := mgmntClient.PostVariantOrder(workspace, slug, "draft", &variantOrder); err != nil {
			return "", "", clierr.Wrap(err, clierr.CodeAPIInternal, fmt.Sprintf("failed to push variant order for template %s", slug))
		}
	}

	if !commit {
		return utils.StatusPushed, "", nil
	}
	return commitTemplate(mgmntClient, workspace, slug, commitMessage, skipInvalid)
}

// commitTemplate commits a template's draft and returns its report status and
// detail. With skipInvalid it commits only the changed variants that pass
// pre-commit validation and names the ones it skipped.
func commitTemplate(mgmntClient *mgmnt.SS_MgmntClient, workspace, slug, commitMessage string, skipInvalid bool) (string, string, error) {
	var variants []map[string]any
	detail := ""
	if skipInvalid {
		validateResp, err := mgmntClient.PreCommitValidate(workspace, slug)
		if err != nil {
			return "", "", clierr.Wrap(err, clierr.CodeAPIInternal, fmt.Sprintf("failed to pre-commit validate template %s", slug))
		}
		if !validateResp.HasChanges {
			return utils.StatusUnchanged, "no changes to commit", nil
		}
		valid, skipped, err := selectValidVariants(validateResp, slug)
		if err != nil {
			return "", "", err
		}
		if len(skipped) > 0 {
			variants = valid
			detail = fmt.Sprintf("skipped %d invalid variant(s): %s", len(skipped), strings.Join(skipped, ", "))
		}
	}

	status, err := mgmntClient.CommitTemplate(workspace, slug, commitMessage, variants)
	if err != nil {
		hint := ""
		if !skipInvalid {
			hint = "to commit only the variants that pass validation, add --skip-invalid"
		}
		return "", "", withHint(clierr.Wrap(err, clierr.CodeAPIInternal, ""), hint)
	}
	if status == mgmnt.NoChanges {
		return utils.StatusUnchanged, "no changes to commit", nil
	}
	return utils.StatusCommitted, detail, nil
}

// withHint sets hint on err unless it already has one.
func withHint(err *clierr.CLIError, hint string) error {
	if err.Hint == "" {
		err.Hint = hint
	}
	return err
}

// templateFields picks the fields of template.json that the template upsert
// accepts. A field that is missing or null in the file is left out so the
// template keeps its current value: sending "enabled_channels": null clears the
// draft's channels. An empty name is left out because the API rejects it.
func templateFields(templateData map[string]any) map[string]any {
	fields := map[string]any{}
	for _, key := range mgmnt.TemplateFields {
		if v, ok := templateData[key]; ok && v != nil {
			fields[key] = v
		}
	}
	if name, ok := fields["name"].(string); ok && strings.TrimSpace(name) == "" {
		delete(fields, "name")
	}
	return fields
}

// mockDataIDFields are the mock data fields the API is known to drop without
// an error (it stores null instead).
var mockDataIDFields = []string{"recipient_distinct_id", "actor_distinct_id", "tenant_id"}

// warnDroppedMockData warns about mock_data.json fields that the API didn't
// store as sent.
func warnDroppedMockData(slug string, sent, stored map[string]any) {
	if stored == nil {
		return
	}
	for _, key := range mockDataIDFields {
		want, ok := sent[key]
		if !ok || want == nil || want == "" {
			continue
		}
		if got := stored[key]; fmt.Sprint(got) != fmt.Sprint(want) {
			log.Warnf("templates/%s: mock_data.json sets %s to %v, but the API stored %v", slug, key, want, got)
		}
	}
}

var templatePushCmd = &cobra.Command{
	Use:   "push [<slug>]",
	Short: "Push templates and their variants from local to SuprSend workspace",
	Long:  "Push templates and their variants from local to SuprSend workspace. Pass a slug as a positional argument or via --slug to push a single template, or omit to push all.\n\n" + templateLayoutHelp,
	Example: `  # Push all templates from default directory
  suprsend template push

  # Push a single template and commit to live immediately
  suprsend template push welcome-email --commit

  # Commit only the variants that pass validation
  suprsend template push welcome-email --commit --skip-invalid

  # Dry run: preview what would be pushed without making changes
  suprsend template push --dry-run`,
	Annotations: map[string]string{
		"skills:tip.a-draft":   "Push writes to the **draft** state. Run `suprsend template commit` to promote draft → live, or pass `--commit` to push + commit in one step.",
		"skills:tip.b-dryrun":  "`--dry-run` makes no API calls: it only reads the local template directories and reports the variants and channels that would be pushed. Nothing is validated server-side.",
		"skills:tip.c-invalid": "With `--commit`, all changed variants are committed together, and one invalid variant fails the commit. `--skip-invalid` runs a pre-commit validation and commits only the variants that pass; the skipped ones are listed in the result. If every changed variant fails, nothing is committed and push exits non-zero; if nothing changed, the template is reported as `unchanged`.",
		"skills:tip.d-confirm": "The `--commit` confirmation prompt only appears when stdin is a TTY. In non-interactive shells (CI, agents) it proceeds without asking; `-F/--force` skips it on a TTY too.",
		"skills:tip.e-result":  "Each template is reported on stdout as `pushed`, `committed`, `unchanged` (the draft already matched live) or `failed`, followed by a summary; with `-o json` the result is one JSON object with `items` and `counts`.",
	},
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		path, _ := cmd.Flags().GetString("dir")
		commit, _ := cmd.Flags().GetBool("commit")
		commitMessage, _ := cmd.Flags().GetString("commit-message")
		slug := utils.ResolveSlug(cmd, args)
		force, _ := cmd.Flags().GetBool("force")
		skipInvalid, _ := cmd.Flags().GetBool("skip-invalid")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		if skipInvalid && !commit {
			return clierr.New("--skip-invalid only applies with --commit", clierr.CodeInvalidUsage)
		}

		if commit && !dryRun && !force {
			target := "all templates"
			if slug != "" {
				target = fmt.Sprintf("template '%s'", slug)
			}
			msg := fmt.Sprintf("This will push and promote %s to live in workspace \"%s\". Continue?", target, workspace)
			confirmed, err := utils.ConfirmDestructiveAction(msg)
			if err != nil || !confirmed {
				log.Info("Aborted.")
				return nil
			}
		}

		if path == "" {
			path = filepath.Join(".", "suprsend", "templates")
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			return clierr.Wrap(err, clierr.CodeFileNotFound, "")
		}

		mgmntClient := utils.GetSuprSendMgmntClient()
		report := utils.NewReport("template push", workspace, dryRun)

		if slug != "" {
			templateDir := filepath.Join(path, slug)
			if _, err := os.Stat(templateDir); os.IsNotExist(err) {
				return clierr.Wrap(err, clierr.CodeFileNotFound, fmt.Sprintf("template directory %s does not exist", templateDir))
			}

			spinner := utils.NewSpinner(fmt.Sprintf("Pushing template %s...", slug))
			status, detail, err := PushTemplate(mgmntClient, workspace, slug, templateDir, commitMessage, commit, skipInvalid, dryRun)
			spinner.Stop("")
			if err != nil {
				report.Fail("templates/"+slug, err)
				return report.Finish(err)
			}
			report.Add("templates/"+slug, status, detail)
			return report.Finish(nil)
		}

		entries, err := os.ReadDir(path)
		if err != nil {
			return clierr.Wrap(err, clierr.CodeFileNotFound, "")
		}

		var failures []string
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			templateSlug := entry.Name()
			templateDir := filepath.Join(path, templateSlug)

			spinner := utils.NewSpinner(fmt.Sprintf("Pushing template %s...", templateSlug))
			status, detail, err := PushTemplate(mgmntClient, workspace, templateSlug, templateDir, commitMessage, commit, skipInvalid, dryRun)
			spinner.Stop("")
			if err != nil {
				report.Fail("templates/"+templateSlug, err)
				failures = append(failures, clierr.ItemFailure("templates/"+templateSlug, "failed to push", err))
				continue
			}
			report.Add("templates/"+templateSlug, status, detail)
		}

		if len(failures) > 0 {
			code := clierr.CodeAPIInternal
			summary := fmt.Sprintf("%d template(s) failed to push", len(failures))
			if dryRun {
				code = clierr.CodeFileParseFailed
				summary = fmt.Sprintf("%d template(s) failed local checks", len(failures))
			}
			return report.Finish(clierr.Failures(summary, failures, code))
		}
		return report.Finish(nil)
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
	templatePushCmd.PersistentFlags().BoolP("force", "F", false, "Skip the confirmation prompt for --commit")
	templatePushCmd.PersistentFlags().Bool("skip-invalid", false, "With --commit: commit only the variants that pass pre-commit validation and list the ones skipped")
	templatePushCmd.PersistentFlags().BoolP("dry-run", "n", false, "Report what would be pushed without making any changes")
	TemplateCmd.AddCommand(templatePushCmd)
}
