package mgmnt

import (
	"encoding/json"
	"testing"
)

func TestFlattenMessage(t *testing.T) {
	tests := map[string]string{
		`"plain"`: "plain",
		`{"category":["unknown category"],"name":["required"]}`: "category: unknown category; name: required",
		`["a","b"]`: "a; b",
		`null`:      "",
	}
	for raw, want := range tests {
		if got := flattenMessage(json.RawMessage(raw)); got != want {
			t.Errorf("flattenMessage(%s) = %q, want %q", raw, got, want)
		}
	}
}
