package category

import (
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
	commitStatus, err := client.PushCategories(workspace, categories, commit, commitMessage)
	if err != nil {
		return "", "", err
	}
	if commit {
		s, d := utils.PushStatus(commitStatus)
		return s, d, nil
	}
	if !exists {
		return utils.StatusCreated, "", nil
	}
	return utils.StatusUpdated, "", nil
}
