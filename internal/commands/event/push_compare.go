package event

import (
	"strings"

	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/state"
	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/cli/mgmnt"
)

// eventKey is how the API identifies an event: it stores names in upper case.
func eventKey(obj map[string]any) string {
	name, _ := obj["name"].(string)
	return strings.ToUpper(name)
}

// fingerprint covers the fields an event push sets.
func fingerprint(obj map[string]any) string {
	description, _ := obj["description"].(string)
	return state.Fingerprint(map[string]any{
		"name":           eventKey(obj),
		"description":    description,
		"payload_schema": schemaLink(obj),
	})
}

// schemaLink is the event's payload_schema the way the API stores it: nil
// when there is no schema slug, else the lower-cased slug with a version_no
// that is null when not pinned. A file that leaves version_no out then
// compares equal to what the API returns.
func schemaLink(obj map[string]any) any {
	ps, _ := obj["payload_schema"].(map[string]any)
	slug, _ := ps["schema"].(string)
	if strings.TrimSpace(slug) == "" {
		return nil
	}
	return map[string]any{"schema": strings.ToLower(strings.TrimSpace(slug)), "version_no": ps["version_no"]}
}

// planEvents compares local events with the workspace's and returns the
// report status for each (created, updated or unchanged) and the events
// that need sending, in their original order. unlink names the events whose
// schema link the local copy removes: the bulk endpoint clears the event's
// payload_schema but, in a request where no event has a schema, leaves the
// link record that still drives validation, so those are delinked first.
func planEvents(client *mgmnt.SS_MgmntClient, workspace string, events []any) (statuses []string, toSend []any, unlink []string, err error) {
	resp, err := client.GetEvents(workspace)
	if err != nil {
		return nil, nil, nil, clierr.Wrap(err, clierr.CodeAPIInternal, "couldn't list the workspace's events")
	}
	remote := map[string]string{}
	linked := map[string]bool{}
	for _, e := range resp.Results {
		if obj, ok := e.(map[string]any); ok {
			remote[eventKey(obj)] = fingerprint(obj)
			linked[eventKey(obj)] = schemaLink(obj) != nil
		}
	}
	statuses = make([]string, len(events))
	for i, e := range events {
		obj, _ := e.(map[string]any)
		fp, exists := remote[eventKey(obj)]
		switch {
		case !exists:
			statuses[i] = utils.StatusCreated
		case fp == fingerprint(obj):
			statuses[i] = utils.StatusUnchanged
			continue
		default:
			statuses[i] = utils.StatusUpdated
			if linked[eventKey(obj)] && schemaLink(obj) == nil {
				unlink = append(unlink, eventKey(obj))
			}
		}
		toSend = append(toSend, e)
	}
	return statuses, toSend, unlink, nil
}
