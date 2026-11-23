package template

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/state"
	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/cli/mgmnt"
)

// A template is pushed in parts, each compared and recorded on its own, so a
// variant edited in the dashboard isn't overwritten by a push that only
// changed another variant. State file keys are "<slug>/<part>".
const (
	partFields       = "fields" // name, description, tags, enabled_channels
	partMockData     = "mock_data"
	partVariantOrder = "variant_order"
	variantPrefix    = "variants/" // + "<channel>/<id>"
)

// variantIgnore are the variant fields the API manages. Fingerprints leave
// them out, so a draft and its live version (and files pulled in either
// mode) compare equal when their content does.
var variantIgnore = []string{
	"$schema", "action", "approval_at", "approval_status", "created_at", "discard_comment",
	"errors", "has_error", "hash", "needs_vendor_approval", "sysgen_template_name",
	"updated_at", "vendor_approvals",
}

// templateParts maps part keys to their values.
type templateParts map[string]any

func variantKey(v map[string]any) string {
	ch, _ := v["channel"].(string)
	id, _ := v["id"].(string)
	return variantPrefix + ch + "/" + id
}

// localParts are the parts push would send from a template directory.
func localParts(templateData map[string]any, variants []map[string]any, mockData map[string]any) templateParts {
	p := templateParts{partFields: templateFields(templateData)}
	for _, v := range variants {
		p[variantKey(v)] = v
	}
	if mockData != nil {
		p[partMockData] = mockData
	}
	if vo, ok := templateData["variant_order"]; ok && vo != nil {
		p[partVariantOrder] = vo
	}
	return p
}

// remoteParts are the parts of a template as the API returned it.
func remoteParts(t TemplateResult) templateParts {
	tags := t.Tags
	if tags == nil {
		tags = []string{}
	}
	p := templateParts{partFields: map[string]any{
		"name": t.Name, "description": t.Description, "tags": tags, "enabled_channels": t.EnabledChannels,
	}}
	for _, v := range t.Variants {
		p[variantKey(v)] = v
	}
	if t.MockData != nil {
		p[partMockData] = t.MockData
	}
	if t.VariantOrder != nil {
		p[partVariantOrder] = flattenVariantOrder(t.VariantOrder)
	}
	return p
}

// partFingerprint fingerprints one part. The fields part is compared on the
// fields template.json sets (fieldKeys): a field left out keeps the
// workspace's value, so it can't differ.
func partFingerprint(key string, v any, fieldKeys []string) string {
	switch {
	case key == partFields:
		m, _ := v.(map[string]any)
		picked := map[string]any{}
		for _, k := range fieldKeys {
			picked[k] = m[k]
		}
		return state.Fingerprint(picked)
	case strings.HasPrefix(key, variantPrefix):
		m, _ := v.(map[string]any)
		return state.Fingerprint(m, variantIgnore...)
	}
	return state.Fingerprint(v)
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// partKeys lists p's keys with the fields part first, as they are pushed.
func (p templateParts) partKeys() []string {
	keys := []string{}
	if _, ok := p[partFields]; ok {
		keys = append(keys, partFields)
	}
	for _, k := range sortedKeys(p) {
		if k != partFields {
			keys = append(keys, k)
		}
	}
	return keys
}

func (p templateParts) fieldKeys() []string {
	m, _ := p[partFields].(map[string]any)
	return sortedKeys(m)
}

type partPlan struct {
	key           string
	action        state.Action
	local, remote string // fingerprints; remote is "" when the workspace lacks the part
}

type plan []partPlan

func (p plan) keys(a state.Action) []string {
	var keys []string
	for _, pp := range p {
		if pp.action == a {
			keys = append(keys, pp.key)
		}
	}
	return keys
}

// fetchDraft fetches the workspace's draft of slug; exists is false when the
// workspace has no such template. Unlike a pull, any failed request is an
// error: a part that couldn't be read would look deleted in the workspace.
// A 404 for mock data or variant order means the template has none.
func fetchDraft(client *mgmnt.SS_MgmntClient, workspace, slug string) (TemplateResult, bool, error) {
	tmpl, err := client.GetTemplate(workspace, slug, "draft")
	if isNotFound(err) {
		return TemplateResult{}, false, nil
	}
	if err != nil {
		return TemplateResult{}, false, clierr.Wrap(err, clierr.CodeAPIInternal, fmt.Sprintf("couldn't fetch template %s", slug))
	}
	variants, err := client.GetTemplateVariants(workspace, slug, "draft")
	if err != nil {
		return TemplateResult{}, false, clierr.Wrap(err, clierr.CodeAPIInternal, fmt.Sprintf("couldn't fetch variants for template %s", slug))
	}
	mockData, err := client.GetTemplateMockData(workspace, slug)
	if err != nil {
		return TemplateResult{}, false, clierr.Wrap(err, clierr.CodeAPIInternal, fmt.Sprintf("couldn't fetch mock data for template %s", slug))
	}
	variantOrder, err := client.GetVariantOrder(workspace, slug, "draft")
	if isNotFound(err) {
		variantOrder, err = nil, nil
	}
	if err != nil {
		return TemplateResult{}, false, clierr.Wrap(err, clierr.CodeAPIInternal, fmt.Sprintf("couldn't fetch variant order for template %s", slug))
	}
	return TemplateResult{
		Slug:            slug,
		Name:            tmpl.Name,
		Description:     tmpl.Description,
		Tags:            tmpl.Tags,
		EnabledChannels: tmpl.EnabledChannels,
		Variants:        variants,
		MockData:        mockData,
		VariantOrder:    variantOrder,
	}, true, nil
}

func isNotFound(err error) bool {
	var ce *clierr.CLIError
	return errors.As(err, &ce) && ce.Code == clierr.CodeAPINotFound
}

// planParts decides what to do with each local part.
func planParts(client *mgmnt.SS_MgmntClient, workspace, slug string, local templateParts, opts PushOptions) (plan, bool, error) {
	var p plan
	if opts.PushAll {
		for _, key := range local.partKeys() {
			p = append(p, partPlan{key: key, action: state.Update})
		}
		return p, true, nil
	}
	draft, exists, err := fetchDraft(client, workspace, slug)
	if err != nil {
		return nil, false, err
	}
	remote := templateParts{}
	if exists {
		remote = remoteParts(draft)
	}
	fieldKeys := local.fieldKeys()
	for _, key := range local.partKeys() {
		pp := partPlan{key: key, local: partFingerprint(key, local[key], fieldKeys)}
		if v, ok := remote[key]; ok {
			pp.remote = partFingerprint(key, v, fieldKeys)
		}
		base, hasBase := opts.State.Get(workspace, slug+"/"+key)
		pp.action = state.Decide(pp.local, pp.remote, base, hasBase, opts.Overwrite)
		p = append(p, pp)
	}
	return p, exists, nil
}

func toPush(a state.Action) bool { return a == state.Create || a == state.Update }

// applyParts sends the parts the plan says to push and returns the keys it
// sent. Variants are all attempted; a failed one fails the template before
// mock data and variant order are sent.
func applyParts(client *mgmnt.SS_MgmntClient, workspace, slug string, p plan, local templateParts, variantOrder *mgmnt.VariantOrderResponse) ([]string, error) {
	var pushed []string
	var variantErrs []string
	for _, pp := range p {
		if !toPush(pp.action) {
			continue
		}
		switch {
		case pp.key == partFields:
			fields, _ := local[partFields].(map[string]any)
			if err := client.CreateTemplate(workspace, slug, fields); err != nil {
				return pushed, clierr.Wrap(err, clierr.CodeAPIInternal, fmt.Sprintf("failed to create template %s", slug))
			}
		case strings.HasPrefix(pp.key, variantPrefix):
			variant, _ := local[pp.key].(map[string]any)
			if err := client.PushTemplateVariant(workspace, slug, variant); err != nil {
				variantErrs = append(variantErrs, err.Error())
				continue
			}
		}
		if pp.key == partMockData || pp.key == partVariantOrder {
			continue // after the variants
		}
		pushed = append(pushed, pp.key)
	}
	if len(variantErrs) > 0 {
		return pushed, clierr.New(fmt.Sprintf("templates/%s: failed to push %d variant(s): %s", slug, len(variantErrs), strings.Join(variantErrs, "; ")), clierr.CodeAPIInternal)
	}
	for _, pp := range p {
		if !toPush(pp.action) {
			continue
		}
		switch pp.key {
		case partMockData:
			mockData, _ := local[partMockData].(map[string]any)
			stored, err := client.PatchTemplateMockData(workspace, slug, mockData)
			if err != nil {
				return pushed, clierr.Wrap(err, clierr.CodeAPIInternal, fmt.Sprintf("failed to push mock data for template %s", slug))
			}
			warnDroppedMockData(slug, mockData, stored)
		case partVariantOrder:
			if err := client.PostVariantOrder(workspace, slug, "draft", variantOrder); err != nil {
				return pushed, clierr.Wrap(err, clierr.CodeAPIInternal, fmt.Sprintf("failed to push variant order for template %s", slug))
			}
		default:
			continue
		}
		pushed = append(pushed, pp.key)
	}
	return pushed, nil
}

// recordParts records the parts that were pushed or found unchanged, with
// the workspace's version read back after the push. Skipped and conflicting
// parts keep their old entry, so the next push still sees that the
// workspace changed them. When the template can't be read back, the pushed
// parts' entries are dropped.
func recordParts(client *mgmnt.SS_MgmntClient, st *state.File, workspace, slug string, p plan, pushed []string, local templateParts) {
	if st == nil {
		return
	}
	wasPushed := map[string]bool{}
	for _, k := range pushed {
		wasPushed[k] = true
	}
	var after templateParts
	if len(pushed) > 0 {
		if draft, exists, err := fetchDraft(client, workspace, slug); err == nil && exists {
			after = remoteParts(draft)
		} else {
			log.Debugf("templates/%s: couldn't read the template back after pushing: %v", slug, err)
		}
	}
	fieldKeys := local.fieldKeys()
	for _, pp := range p {
		key := slug + "/" + pp.key
		switch {
		case wasPushed[pp.key]:
			if v, ok := after[pp.key]; ok {
				st.Set(workspace, key, state.Entry{Remote: partFingerprint(pp.key, v, fieldKeys), Local: pp.local})
			} else {
				st.Delete(workspace, key, false)
			}
		case pp.action == state.Unchanged:
			remote := pp.remote
			if v, ok := after[pp.key]; ok {
				remote = partFingerprint(pp.key, v, fieldKeys)
			}
			st.Set(workspace, key, state.Entry{Remote: remote, Local: pp.local})
		}
	}
}

// pushStatus is the report status and detail for a push without commit.
func pushStatus(p plan, pushed, skipped []string, exists, pushAll bool) (string, string) {
	if pushAll {
		return utils.StatusPushed, ""
	}
	skipNote := ""
	if len(skipped) > 0 {
		skipNote = "skipped, changed in the workspace since the last pull or push and not locally: " + describeParts(skipped) + " (pull to update the files, or pass --overwrite)"
	}
	join := func(a, b string) string {
		if a == "" || b == "" {
			return a + b
		}
		return a + "; " + b
	}
	switch {
	case len(pushed) > 0 && !exists:
		return utils.StatusCreated, fmt.Sprintf("%d variant(s)", countVariants(p))
	case len(pushed) > 0:
		return utils.StatusUpdated, join(describeParts(pushed), skipNote)
	case len(skipped) > 0:
		return utils.StatusSkipped, skipNote
	}
	return utils.StatusUnchanged, ""
}

func countVariants(p plan) int {
	n := 0
	for _, pp := range p {
		if strings.HasPrefix(pp.key, variantPrefix) {
			n++
		}
	}
	return n
}

// describeParts names parts for a message: email/default, mock data, ...
func describeParts(keys []string) string {
	names := make([]string, 0, len(keys))
	for _, k := range keys {
		switch {
		case k == partFields:
			names = append(names, "template fields")
		case k == partMockData:
			names = append(names, "mock data")
		case k == partVariantOrder:
			names = append(names, "variant order")
		default:
			names = append(names, strings.TrimPrefix(k, variantPrefix))
		}
	}
	const max = 6
	if len(names) > max {
		return fmt.Sprintf("%s and %d more", strings.Join(names[:max], ", "), len(names)-max)
	}
	return strings.Join(names, ", ")
}

// RecordPulled records a template just written by pull to outputDir. The
// local fingerprints come from reading the files back the way push does, so
// a push right after the pull finds nothing changed locally.
func RecordPulled(st *state.File, workspace, outputDir string, t TemplateResult) {
	if st == nil {
		return
	}
	st.Delete(workspace, t.Slug+"/", true)
	local, err := readLocalParts(templateDirOf(outputDir, t.Slug))
	if err != nil {
		log.Debugf("templates/%s: couldn't read back the pulled files: %v", t.Slug, err)
		return
	}
	remote := remoteParts(t)
	fieldKeys := local.fieldKeys()
	for _, key := range local.partKeys() {
		v, ok := remote[key]
		if !ok {
			continue
		}
		st.Set(workspace, t.Slug+"/"+key, state.Entry{
			Remote: partFingerprint(key, v, fieldKeys),
			Local:  partFingerprint(key, local[key], fieldKeys),
		})
	}
}

// saveState writes st, warning (not failing the command) when it can't.
func saveState(st *state.File) {
	if err := st.Save(); err != nil {
		log.Warnf("couldn't save %s: %v; the next push can't tell what changed since this run", state.FileName, err)
	}
}

// readLocalParts reads a template directory the way push does.
func readLocalParts(templateDir string) (templateParts, error) {
	templateData, err := readTemplateJSON(templateDir)
	if err != nil {
		return nil, err
	}
	variants, err := readTemplateVariants(templateDir)
	if err != nil {
		return nil, err
	}
	mockData, err := readMockData(templateDir)
	if err != nil {
		return nil, err
	}
	return localParts(templateData, variants, mockData), nil
}

func templateDirOf(outputDir, slug string) string {
	return filepath.Join(outputDir, slug)
}
