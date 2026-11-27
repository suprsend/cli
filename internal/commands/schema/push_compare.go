package schema

import (
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/state"
	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/cli/mgmnt"
)

// fingerprint covers what a schema push changes: the fields the API reads
// from the request. A new schema is enabled, so a missing is_enabled
// compares as true (sameAs fills it in for a schema that exists). The API
// adds additionalProperties: false to a schema that has properties and
// doesn't set it, so that counts as set on both sides.
func fingerprint(payload map[string]any) string {
	jsonSchema := map[string]any{}
	if js, ok := payload["json_schema"].(map[string]any); ok {
		for k, v := range js {
			if k != "$schema" {
				jsonSchema[k] = v
			}
		}
	}
	if _, ok := jsonSchema["properties"]; ok {
		if _, set := jsonSchema["additionalProperties"]; !set {
			jsonSchema["additionalProperties"] = false
		}
	}
	enabled, ok := payload["is_enabled"].(bool)
	if !ok {
		enabled = true
	}
	return state.Fingerprint(map[string]any{
		"name":        stringOf(payload["name"]),
		"description": stringOf(payload["description"]),
		"json_schema": jsonSchema,
		"is_enabled":  enabled,
	})
}

// sameAs reports whether pushing payload would leave remote as it is. A
// name, description or is_enabled the payload leaves out keeps the stored
// value (as with `schema push --json`, which sends only json_schema), so it
// is taken from remote.
func sameAs(payload, remote map[string]any) bool {
	filled := map[string]any{}
	for k, v := range payload {
		filled[k] = v
	}
	for _, k := range []string{"name", "description", "is_enabled"} {
		if _, ok := filled[k]; !ok {
			filled[k] = remote[k]
		}
	}
	return fingerprint(filled) == fingerprint(remote)
}

func stringOf(v any) string {
	s, _ := v.(string)
	return s
}

// pushSchema pushes a schema unless the workspace's draft already matches
// it, and returns its report status and detail: created, updated,
// unchanged, or with commit committed/unchanged.
func pushSchema(client *mgmnt.SS_MgmntClient, workspace, slug string, payload map[string]any, commit bool, commitMessage string) (string, string, error) {
	remote, err := client.GetSchemaBySlug(workspace, slug, "draft")
	exists := true
	switch {
	case clierr.HasCode(err, clierr.CodeAPINotFound):
		exists = false
	case err != nil:
		return "", "", clierr.Wrap(err, clierr.CodeAPIInternal, "")
	}
	if exists && remote != nil && sameAs(payload, *remote) {
		if !commit {
			return utils.StatusUnchanged, "", nil
		}
		status, err := client.FinalizeSchema(workspace, slug, commitMessage)
		if err != nil {
			return "", "", err
		}
		s, d := utils.PushStatus(status)
		return s, d, nil
	}
	if exists && remote != nil && matchesLive(client, workspace, slug, payload) {
		// The file is the live version (a pull without --mode draft), and
		// the draft differs: it is someone's pending work. Pushing would
		// replace it with what is already live, so leave it alone.
		note := "the workspace's draft has changes that aren't in the file, which matches live; pull with --mode draft to see them"
		if commit {
			note = "not committed: " + note
		}
		return utils.StatusUnchanged, note, nil
	}
	commitStatus, err := client.PushSchema(workspace, slug, payload, commit, commitMessage)
	if err != nil {
		return "", "", err
	}
	if commit {
		s, d := utils.PushStatus(commitStatus)
		if !exists {
			d = "created"
		}
		return s, d, nil
	}
	if !exists {
		return utils.StatusCreated, "", nil
	}
	return utils.StatusUpdated, "", nil
}

// matchesLive reports whether payload is the schema's live version. Any
// failure to fetch it (there may be no live version yet) counts as no.
func matchesLive(client *mgmnt.SS_MgmntClient, workspace, slug string, payload map[string]any) bool {
	live, err := client.GetSchemaBySlug(workspace, slug, "live")
	if err != nil || live == nil {
		return false
	}
	return sameAs(payload, *live)
}
