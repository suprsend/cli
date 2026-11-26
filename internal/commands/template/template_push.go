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
	"github.com/suprsend/cli/internal/state"
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

// PushOptions control how PushTemplate pushes a template.
type PushOptions struct {
	CommitMessage string
	Commit        bool
	// SkipInvalid, with Commit, commits only the variants that pass
	// pre-commit validation; otherwise an invalid variant fails the commit.
	SkipInvalid bool
	DryRun      bool
	// Overwrite pushes the local copy of each part even over changes made in
	// the workspace since the last pull or push.
	Overwrite bool
	// PushAll sends every part without comparing with the workspace first
	// (sync, which makes the destination match the source).
	PushAll bool
	// State is the record of earlier pulls and pushes; nil for none.
	State *state.File
}

// PushTemplate pushes one template directory and, with opts.Commit, promotes
// it to live. Each part (the template's fields, each variant, mock data,
// variant order) is compared with the workspace's draft and with the record
// of the last pull or push, and only the parts changed locally are sent. A
// part changed in the workspace since then is left alone, and one changed on
// both sides fails the template, unless opts.Overwrite is set. It returns the
// report status (created, updated, unchanged, skipped, committed, pushed for
// PushAll, or would-push/would-commit on a dry run) and a detail note.
func PushTemplate(mgmntClient *mgmnt.SS_MgmntClient, workspace, slug, templateDir string, opts PushOptions) (string, string, error) {
	resource := "templates/" + slug
	templateData, err := readTemplateJSON(templateDir)
	if err != nil {
		return "", "", clierr.Wrap(err, clierr.CodeFileParseFailed, fmt.Sprintf("failed to read template.json for template %s", slug))
	}
	variants, err := readTemplateVariants(templateDir)
	if err != nil {
		return "", "", clierr.Wrap(err, clierr.CodeFileParseFailed, fmt.Sprintf("failed to read template %s", slug))
	}
	mockData, err := readMockData(templateDir)
	if err != nil {
		return "", "", clierr.Wrap(err, clierr.CodeFileParseFailed, fmt.Sprintf("failed to parse mock_data.json for template %s", slug))
	}
	var variantOrder *mgmnt.VariantOrderResponse
	if raw, ok := templateData["variant_order"]; ok && raw != nil {
		if variantOrder, err = variantOrderFromFile(raw); err != nil {
			return "", "", clierr.New(fmt.Sprintf("invalid variant_order format for template %s", slug), clierr.CodeFileParseFailed)
		}
	}

	if opts.DryRun {
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
		if opts.Commit {
			status = utils.StatusWouldCommit
		}
		return status, fmt.Sprintf("%d variant(s), channels: %s", len(variants), strings.Join(channels, ", ")), nil
	}

	local := localParts(templateData, variants, mockData)
	plan, remote, exists, err := planParts(mgmntClient, workspace, slug, local, opts)
	if err != nil {
		return "", "", err
	}
	if conflicts := plan.keys(state.Conflict); len(conflicts) > 0 {
		return "", "", clierr.New(fmt.Sprintf("%s: changed in workspace %q since the last pull or push, and also changed locally: %s", resource, workspace, describeParts(conflicts)), clierr.CodeAPIConflict).
			WithResource(resource).
			WithHint("copy your edits, pull the template to get the workspace's version (with --mode draft if you pull live and the change is a draft), and reapply them; or pass --overwrite to replace the workspace's version with yours")
	}
	// Parts whose draft holds changes a live pull left out. A commit would
	// publish them: with nothing else to send, leave the template alone;
	// with other parts to send, refuse rather than publish them unseen.
	if pending := plan.keys(state.Pending); opts.Commit && len(pending) > 0 {
		if len(plan.keys(state.Create))+len(plan.keys(state.Update)) == 0 {
			return utils.StatusUnchanged, "not committed: " + pendingNote(pending), nil
		}
		return "", "", clierr.New(fmt.Sprintf("%s: not committed: %s", resource, pendingNote(pending)), clierr.CodeAPIConflict).
			WithResource(resource)
	}
	skipped := plan.keys(state.Skip)
	if opts.Commit && len(skipped) > 0 {
		return "", "", clierr.New(fmt.Sprintf("%s: not committed: the workspace's draft changed since the last pull or push, and your copy didn't: %s", resource, describeParts(skipped)), clierr.CodeAPIConflict).
			WithResource(resource).
			WithHint("pull the template to see the change before committing, or pass --overwrite to push and commit your copy")
	}

	// A commit publishes the whole draft, including variants the local files
	// don't have. Refuse to publish ones nobody here has seen; with
	// --overwrite they are committed and named in the result.
	var unseen []string
	if opts.Commit && !opts.PushAll {
		if unseen, err = unseenDraftVariants(mgmntClient, workspace, slug, local, remote); err != nil {
			return "", "", err
		}
		if len(unseen) > 0 && !opts.Overwrite {
			return "", "", clierr.New(fmt.Sprintf("%s: not committed: the workspace's draft has variants that aren't in your files and aren't live yet (or differ from live): %s", resource, describeParts(unseen)), clierr.CodeAPIConflict).
				WithResource(resource).
				WithHint("committing would publish them too: pull the template with --mode draft to review them, or pass --overwrite to commit them as they are")
		}
	}

	pushed, pushErr := applyParts(mgmntClient, workspace, slug, plan, local, variantOrder)
	if !opts.PushAll {
		recordParts(mgmntClient, opts.State, workspace, slug, plan, pushed, local)
	}
	if pushErr != nil {
		return "", "", pushErr
	}

	status, detail := pushStatus(plan, pushed, skipped, exists, opts.PushAll)
	if !opts.Commit {
		return status, detail, nil
	}
	commitStatus, commitDetail, err := commitTemplate(mgmntClient, workspace, slug, opts.CommitMessage, opts.SkipInvalid)
	if err != nil {
		return "", "", err
	}
	var notes []string
	if len(pushed) > 0 && !opts.PushAll {
		notes = append(notes, "pushed "+describeParts(pushed))
	}
	if len(unseen) > 0 {
		notes = append(notes, "also committed from the workspace's draft, not in your files: "+describeParts(unseen))
	}
	if commitDetail != "" {
		notes = append(notes, commitDetail)
	}
	return commitStatus, strings.Join(notes, "; "), nil
}

// readMockData reads mock_data.json, or returns nil when there is none.
func readMockData(templateDir string) (map[string]any, error) {
	data, err := os.ReadFile(filepath.Join(templateDir, "mock_data.json"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var mockData map[string]any
	if err := json.Unmarshal(data, &mockData); err != nil {
		return nil, err
	}
	return mockData, nil
}

// variantOrderFromFile turns template.json's variant_order ("<channel>" or
// "<channel>/<tenant>" mapped to variant ids) into the API's form.
func variantOrderFromFile(raw any) (*mgmnt.VariantOrderResponse, error) {
	flatOrder, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("variant_order is not an object")
	}
	channelMap := map[string]*mgmnt.VariantOrderChannel{}
	var channels []string
	for _, key := range sortedKeys(flatOrder) {
		parts := strings.SplitN(key, "/", 2)
		channel := parts[0]
		var tenantIDPtr *string
		if len(parts) == 2 {
			t := parts[1]
			tenantIDPtr = &t
		}
		variants := []string{} // the API rejects null
		if variantsList, ok := flatOrder[key].([]any); ok {
			for _, v := range variantsList {
				if s, ok := v.(string); ok {
					variants = append(variants, s)
				}
			}
		}
		if _, exists := channelMap[channel]; !exists {
			channelMap[channel] = &mgmnt.VariantOrderChannel{Channel: channel}
			channels = append(channels, channel)
		}
		channelMap[channel].Tenants = append(channelMap[channel].Tenants, mgmnt.VariantOrderTenant{
			TenantID: tenantIDPtr,
			Variants: variants,
		})
	}
	// The API rejects null channels, so send [] for a template with no variants.
	order := &mgmnt.VariantOrderResponse{Channels: []mgmnt.VariantOrderChannel{}}
	for _, ch := range channels {
		order.Channels = append(order.Channels, *channelMap[ch])
	}
	return order, nil
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
		"skills:tip.e-result":  "Each template is reported on stdout as `created`, `updated` (the detail names the parts sent, such as `email/default` or `mock data`), `unchanged` (nothing differed from the workspace's draft, so nothing was sent; with `--commit`, the draft already matched live), `skipped` (see the state tip), `committed` or `failed`, followed by a summary; with `-o json` the result is one JSON object with `items` and `counts`.",
		"skills:tip.f-state":   "A template is pushed in parts: its fields (name, description, tags, enabled_channels), each variant, mock data and variant order, and only the parts that differ from the workspace's draft are sent. Pull and push keep `.suprsend-state.json` in the templates directory (commit it with the files): what each part looked like, per workspace, at the last pull or push. A part changed only in the workspace since then (say, a variant edited in the dashboard) is skipped rather than overwritten, and named in the detail; with `--commit` that is an error, so nobody else's draft goes live unseen. For the same reason `--commit` fails when the workspace's draft has a variant that isn't in your files and is new or differs from live (a commit publishes the whole draft): pull with `--mode draft` to review it, or pass `--overwrite` to commit it as it is (the result names it). A part changed on both sides fails the template as a conflict (code `api-conflict`) before anything is sent. Pull again to get the workspace's version, or pass `--overwrite` to push your copy anyway. Without a record for that workspace, every part that differs is sent. A pull takes the live version by default, while push compares with the draft. Parts whose draft differed from live at the pull are someone's pending work: push leaves them alone and names them in the detail, a local edit to one fails as a conflict, and `--commit` does not publish them (with nothing else to send the template is `unchanged, not committed`; with other parts to send it fails). Pull with `--mode draft` to work on the draft, or pass `--overwrite` to replace it.",
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
		overwrite, _ := cmd.Flags().GetBool("overwrite")

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
		opts := PushOptions{CommitMessage: commitMessage, Commit: commit, SkipInvalid: skipInvalid, DryRun: dryRun, Overwrite: overwrite}
		if !dryRun {
			st, err := state.Load(path)
			if err != nil {
				return err
			}
			opts.State = st
		}
		report := utils.NewReport("template push", workspace, dryRun)

		if slug != "" {
			templateDir := filepath.Join(path, slug)
			if _, err := os.Stat(templateDir); os.IsNotExist(err) {
				return clierr.Wrap(err, clierr.CodeFileNotFound, fmt.Sprintf("template directory %s does not exist", templateDir))
			}

			spinner := utils.NewSpinner(fmt.Sprintf("Pushing template %s...", slug))
			status, detail, err := PushTemplate(mgmntClient, workspace, slug, templateDir, opts)
			spinner.Stop("")
			saveState(opts.State)
			if err != nil {
				report.Fail("templates/"+slug, err)
				return report.Finish(err)
			}
			report.Add("templates/"+slug, status, detail)
			return report.Finish(nil)
		}

		entries, err := utils.ReadDir(path)
		if err != nil {
			return clierr.Wrap(err, clierr.CodeFileNotFound, "")
		}

		var failures []string
		var failedErrs []error
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			templateSlug := entry.Name()
			templateDir := filepath.Join(path, templateSlug)

			spinner := utils.NewSpinner(fmt.Sprintf("Pushing template %s...", templateSlug))
			status, detail, err := PushTemplate(mgmntClient, workspace, templateSlug, templateDir, opts)
			spinner.Stop("")
			if err != nil {
				report.Fail("templates/"+templateSlug, err)
				failures = append(failures, clierr.ItemFailure("templates/"+templateSlug, "failed to push", err))
				failedErrs = append(failedErrs, err)
				continue
			}
			report.Add("templates/"+templateSlug, status, detail)
		}

		saveState(opts.State)
		if len(failures) > 0 {
			code := clierr.FailuresCode(failedErrs)
			summary := fmt.Sprintf("%d template(s) failed to push", len(failures))
			if code == clierr.CodeFileParseFailed {
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
		if path != templateDir && utils.IsHidden(d.Name()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
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
	templatePushCmd.PersistentFlags().Bool("overwrite", false, "Push your copy of each part even when the workspace's version changed since the last pull or push, replacing that change")
	TemplateCmd.AddCommand(templatePushCmd)
}
