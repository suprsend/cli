package category

import (
	"strings"

	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/state"
	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/cli/mgmnt"
)

// pushCategories pushes the categories unless the workspace's draft already
// matches them, and returns the report status and detail: created, updated,
// unchanged, or with commit committed/unchanged.
func pushCategories(client *mgmnt.SS_MgmntClient, workspace string, categories *CategoriesOnDisk, commit bool, commitMessage string) (string, string, error) {
	remote, err := client.ListCategories(workspace, "draft")
	exists := true
	switch {
	case clierr.HasCode(err, clierr.CodeAPINotFound):
		exists = false
	case err != nil:
		return "", "", clierr.Wrap(err, clierr.CodeAPIInternal, "")
	}
	if exists && remote != nil && state.Fingerprint(remote.RootCategories) == state.Fingerprint(categories.RootCategories) {
		if !commit {
			return utils.StatusUnchanged, "", nil
		}
		status, err := client.FinalizeCategories(workspace, commitMessage)
		if err != nil {
			return "", "", err
		}
		s, d := utils.PushStatus(status)
		return s, d, nil
	}
	if exists && remote != nil {
		// The file is the live tree (a pull without --mode draft) and the
		// draft differs: it is someone's pending work. A push replaces the
		// whole draft tree, so leave it alone.
		if live, err := client.ListCategories(workspace, "live"); err == nil && live != nil &&
			state.Fingerprint(live.RootCategories) == state.Fingerprint(categories.RootCategories) {
			note := "the workspace's draft has changes that aren't in the file, which matches live; pull with --mode draft to see them"
			if commit {
				note = "not committed: " + note
			}
			return utils.StatusUnchanged, note, nil
		}
	}
	commitStatus, err := client.PushCategories(workspace, categories, commit, commitMessage)
	if err != nil {
		return "", "", err
	}
	if commit {
		s, d := utils.PushStatus(commitStatus)
		if note := removedNote(remote, categories); note != "" && exists {
			d = strings.TrimPrefix(d+"; "+note, "; ")
		}
		return s, d, nil
	}
	if !exists {
		return utils.StatusCreated, "", nil
	}
	return utils.StatusUpdated, removedNote(remote, categories), nil
}

// removedNote names the categories a push took out of the draft. A push
// replaces the whole tree, so a category missing from the file is deleted.
func removedNote(remote *mgmnt.PreferenceCategoryResponse, categories *CategoriesOnDisk) string {
	if remote == nil || categories == nil {
		return ""
	}
	removed := RemovedCategories(remote.RootCategories, categories.RootCategories)
	if len(removed) == 0 {
		return ""
	}
	return "removed from the draft, not in the file: " + strings.Join(removed, ", ")
}

// RemovedCategories lists the category slugs in before that after lacks.
func RemovedCategories(before, after []mgmnt.RootCategory) []string {
	kept := map[string]bool{}
	for _, slug := range categorySlugs(after) {
		kept[slug] = true
	}
	var removed []string
	for _, slug := range categorySlugs(before) {
		if !kept[slug] {
			removed = append(removed, slug)
		}
	}
	return removed
}

func categorySlugs(roots []mgmnt.RootCategory) []string {
	var slugs []string
	for _, root := range roots {
		for _, section := range root.Sections {
			for _, c := range section.Categories {
				if c.Category != "" {
					slugs = append(slugs, c.Category)
				}
			}
		}
	}
	return slugs
}
