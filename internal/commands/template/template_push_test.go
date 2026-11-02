package template

import (
	"reflect"
	"testing"

	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/cli/mgmnt"
)

// template.json's name, description and tags are pushed; a field missing
// from the file is left out so the template keeps its current value.
func TestTemplateFields(t *testing.T) {
	got := templateFields(map[string]any{
		"$schema":          templateJSONSchema,
		"slug":             "welcome",
		"name":             "Welcome",
		"description":      "",
		"tags":             []any{"onboarding"},
		"enabled_channels": []any{"email"},
		"variant_order":    map[string]any{},
	})
	want := map[string]any{
		"name":             "Welcome",
		"description":      "",
		"tags":             []any{"onboarding"},
		"enabled_channels": []any{"email"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("templateFields = %#v, want %#v", got, want)
	}

	got = templateFields(map[string]any{"name": "  "})
	if _, ok := got["name"]; ok {
		t.Errorf("an empty name must not be sent: %#v", got)
	}
	if _, ok := got["description"]; ok {
		t.Errorf("a missing description must not be sent: %#v", got)
	}
	// Sending "enabled_channels": null would clear the draft's channels.
	if _, ok := got["enabled_channels"]; ok {
		t.Errorf("missing enabled_channels must not be sent: %#v", got)
	}

	// template pull writes null when the API returns no channels.
	got = templateFields(map[string]any{"enabled_channels": nil})
	if _, ok := got["enabled_channels"]; ok {
		t.Errorf("null enabled_channels must not be sent: %#v", got)
	}

	got = templateFields(map[string]any{"enabled_channels": []any{}})
	if !reflect.DeepEqual(got, map[string]any{"enabled_channels": []any{}}) {
		t.Errorf("an explicit empty enabled_channels must be sent: %#v", got)
	}
}

func TestSelectValidVariants(t *testing.T) {
	resp := &mgmnt.PreCommitValidateResponse{HasChanges: true}
	if err := utils.Remarshal([]any{
		map[string]any{"channel": "email", "id": "v1"},
		map[string]any{"channel": "sms", "id": "v2", "errors": map[string]any{"body": []any{"is empty"}}},
	}, &resp.Variants); err != nil {
		t.Fatal(err)
	}
	valid, skipped, err := selectValidVariants(resp, "welcome")
	if err != nil {
		t.Fatal(err)
	}
	if len(valid) != 1 || valid[0]["id"] != "v1" {
		t.Errorf("valid = %v, want only email/v1", valid)
	}
	if !reflect.DeepEqual(skipped, []string{"sms/v2"}) {
		t.Errorf("skipped = %v, want [sms/v2]", skipped)
	}
}
