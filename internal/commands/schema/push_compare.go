package schema

import (
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/state"
	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/cli/mgmnt"
)

// fingerprint covers what a schema push changes: the fields the API reads
// from the request. A push without is_enabled enables the schema, so a
// missing is_enabled compares as true.
func fingerprint(payload map[string]any) string {
	jsonSchema := map[string]any{}
	if js, ok := payload["json_schema"].(map[string]any); ok {
		for k, v := range js {
			if k != "$schema" {
				jsonSchema[k] = v
			}
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
	if exists && remote != nil && fingerprint(*remote) == fingerprint(payload) {
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
