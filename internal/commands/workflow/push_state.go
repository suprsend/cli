package workflow

import (
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/state"
	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/cli/mgmnt"
)

// workflowIgnore are the workflow fields the API manages. Fingerprints leave
// them out, so a draft and its live version (and files pulled in either
// mode) compare equal when their definitions do.
var workflowIgnore = []string{
	"$schema", "active_at", "commit_message", "created_at", "hash",
	"last_executed_at", "status", "updated_at", "updated_by", "validation_result",
}

func fingerprint(wf map[string]any) string {
	return state.Fingerprint(wf, workflowIgnore...)
}

// RecordPulled records a workflow just written by pull: the file and the
// workspace hold the same definition.
func RecordPulled(st *state.File, workspace string, wf map[string]any) {
	slug, _ := wf["slug"].(string)
	if slug == "" {
		return
	}
	fp := fingerprint(wf)
	st.Set(workspace, slug, state.Entry{Remote: fp, Local: fp})
}

// draftFingerprint fingerprints the workspace's draft of slug, or returns ""
// when the workspace has no such workflow.
func draftFingerprint(client *mgmnt.SS_MgmntClient, workspace, slug string) (string, error) {
	wf, err := client.GetWorkflowDetailBySlug(workspace, slug, "draft")
	var ce *clierr.CLIError
	if errors.As(err, &ce) && ce.Code == clierr.CodeAPINotFound {
		return "", nil
	}
	if err != nil {
		return "", clierr.Wrap(err, clierr.CodeAPIInternal, "")
	}
	if wf == nil {
		return "", nil
	}
	return fingerprint(*wf), nil
}

// pushOptions control how one workflow is pushed.
type pushOptions struct {
	commit        bool
	commitMessage string
	// overwrite pushes the local copy even over changes made in the
	// workspace since the last pull or push.
	overwrite bool
	// state is the record of earlier pulls and pushes; nil when there is
	// no directory to keep one in (--json).
	state *state.File
}

// pushOne pushes a workflow unless the workspace already has it, or has a
// newer version of it, and returns its report status and detail.
func pushOne(client *mgmnt.SS_MgmntClient, workspace, slug string, wf map[string]any, opts pushOptions) (string, string, error) {
	resource := "workflows/" + slug
	local := fingerprint(wf)
	remote, err := draftFingerprint(client, workspace, slug)
	if err != nil {
		return "", "", err
	}
	base, hasBase := opts.state.Get(workspace, slug)

	switch action := state.Decide(local, remote, base, hasBase, opts.overwrite); action {
	case state.Conflict:
		return "", "", conflictError(resource, workspace, remote == "")
	case state.Skip:
		if opts.commit {
			return "", "", clierr.New(fmt.Sprintf("%s: not committed: %s", resource, skipReason(workspace, remote == "")), clierr.CodeAPIConflict).
				WithResource(resource).
				WithHint("pull it to see the change before committing, or pass --overwrite to push and commit your copy")
		}
		return utils.StatusSkipped, skipReason(workspace, remote == "") + "; pull to update the file, or pass --overwrite", nil
	case state.Unchanged:
		opts.state.Set(workspace, slug, state.Entry{Remote: remote, Local: local})
		if !opts.commit {
			return utils.StatusUnchanged, "", nil
		}
		status, err := client.FinalizeWorkflow(workspace, slug, opts.commitMessage)
		if err != nil {
			return "", "", err
		}
		s, d := utils.PushStatus(status)
		return s, d, nil
	default:
		commitStatus, err := client.PushWorkflow(workspace, slug, wf, opts.commit, opts.commitMessage)
		if err == nil || mgmnt.DraftSaved(err) {
			recordPushed(client, opts.state, workspace, slug, local)
		}
		if err != nil {
			return "", "", err
		}
		if opts.commit {
			s, d := utils.PushStatus(commitStatus)
			if action == state.Create {
				d = "created"
			}
			return s, d, nil
		}
		if action == state.Create {
			return utils.StatusCreated, "", nil
		}
		return utils.StatusUpdated, "", nil
	}
}

// recordPushed records what the workspace holds after a push. When it can't
// be read back, the entry is dropped, so the next push compares with the
// workspace alone rather than with a stale record.
func recordPushed(client *mgmnt.SS_MgmntClient, st *state.File, workspace, slug, local string) {
	if st == nil {
		return
	}
	remote, err := draftFingerprint(client, workspace, slug)
	if err != nil || remote == "" {
		st.Delete(workspace, slug, false)
		return
	}
	st.Set(workspace, slug, state.Entry{Remote: remote, Local: local})
}

// saveState writes st, warning (not failing the push) when it can't: the
// push itself went through.
func saveState(st *state.File) {
	if err := st.Save(); err != nil {
		log.Warnf("couldn't save %s: %v; the next push can't tell what changed since this one", state.FileName, err)
	}
}

func skipReason(workspace string, deleted bool) string {
	if deleted {
		return fmt.Sprintf("deleted in workspace %q since the last pull or push, and not changed locally", workspace)
	}
	return fmt.Sprintf("changed in workspace %q since the last pull or push, and not changed locally", workspace)
}

func conflictError(resource, workspace string, deleted bool) error {
	what := "changed"
	if deleted {
		what = "deleted"
	}
	return clierr.New(fmt.Sprintf("%s: %s in workspace %q since the last pull or push, and also changed locally", resource, what, workspace), clierr.CodeAPIConflict).
		WithResource(resource).
		WithHint("copy your edit, pull to get the workspace's version, and reapply it; or pass --overwrite to replace the workspace's version with yours")
}
