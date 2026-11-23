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
		"payload_schema": obj["payload_schema"],
	})
}

// planEvents compares local events with the workspace's and returns the
// report status for each (created, updated or unchanged) and the events
// that need sending, in their original order.
func planEvents(client *mgmnt.SS_MgmntClient, workspace string, events []any) ([]string, []any, error) {
	resp, err := client.GetEvents(workspace)
	if err != nil {
		return nil, nil, clierr.Wrap(err, clierr.CodeAPIInternal, "couldn't list the workspace's events")
	}
	remote := map[string]string{}
	for _, e := range resp.Results {
		if obj, ok := e.(map[string]any); ok {
			remote[eventKey(obj)] = fingerprint(obj)
		}
	}
	statuses := make([]string, len(events))
	var toSend []any
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
		}
		toSend = append(toSend, e)
	}
	return statuses, toSend, nil
}
