package mcpsdk

import (
	"encoding/json"
	"fmt"
)

// Args wraps a tool call's raw JSON arguments. The helper methods give handler
// code a small, runtime-agnostic surface for pulling typed values out of a tool
// call regardless of which adapter registered the tool.
type Args struct {
	raw    json.RawMessage
	parsed map[string]any
}

// NewArgs constructs an Args from raw JSON. An empty or null input is treated
// as an empty object.
func NewArgs(raw json.RawMessage) (Args, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return Args{raw: raw, parsed: map[string]any{}}, nil
	}
	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return Args{}, fmt.Errorf("mcpsdk: invalid tool arguments: %w", err)
	}
	return Args{raw: raw, parsed: parsed}, nil
}

// Raw returns the underlying JSON bytes.
func (a Args) Raw() json.RawMessage { return a.raw }

// Map returns the decoded argument map.
func (a Args) Map() map[string]any { return a.parsed }

// RequireString returns the named argument as a string, or an error if it is
// missing or not a string.
func (a Args) RequireString(name string) (string, error) {
	v, ok := a.parsed[name]
	if !ok {
		return "", fmt.Errorf("required argument %q not found", name)
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("argument %q must be a string", name)
	}
	return s, nil
}

// GetString returns the named argument as a string, or def if it is missing
// or not a string.
func (a Args) GetString(name, def string) string {
	v, ok := a.parsed[name]
	if !ok {
		return def
	}
	s, ok := v.(string)
	if !ok {
		return def
	}
	return s
}

// GetInt returns the named argument as an int, or def if it is missing or not
// a number. JSON numbers decode to float64, so this truncates.
func (a Args) GetInt(name string, def int) int {
	v, ok := a.parsed[name]
	if !ok {
		return def
	}
	f, ok := v.(float64)
	if !ok {
		return def
	}
	return int(f)
}

// GetBool returns the named argument as a bool, or def if missing.
func (a Args) GetBool(name string, def bool) bool {
	v, ok := a.parsed[name]
	if !ok {
		return def
	}
	b, ok := v.(bool)
	if !ok {
		return def
	}
	return b
}

// GetMap returns the named argument as a map, or nil/false if missing or not
// an object.
func (a Args) GetMap(name string) (map[string]any, bool) {
	v, ok := a.parsed[name]
	if !ok {
		return nil, false
	}
	m, ok := v.(map[string]any)
	return m, ok
}

// GetArray returns the named argument as a slice, or nil/false if missing or
// not an array.
func (a Args) GetArray(name string) ([]any, bool) {
	v, ok := a.parsed[name]
	if !ok {
		return nil, false
	}
	arr, ok := v.([]any)
	return arr, ok
}
