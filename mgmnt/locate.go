package mgmnt

import (
	"fmt"
	"sort"
	"strings"
)

// fieldOwners says which objects need a field that the API reports missing
// only by name (a commit error of just 'variable_ns'): an object that has
// every listed key needs the field too. variable_ns belongs to a condition,
// {"variable": ..., "op": ..., "value": ..., "variable_ns": ...}.
var fieldOwners = map[string][]string{
	"variable_ns": {"variable", "op"},
}

// maxPaths caps how many locations an error lists.
const maxPaths = 5

// missingFieldPaths returns the JSON paths (such as
// tree.nodes[1].branches[0].conditions[0]) of the objects in def that need
// field but don't have it. It returns nil for a field it knows nothing about.
func missingFieldPaths(def any, field string) []string {
	owner, ok := fieldOwners[field]
	if !ok || def == nil {
		return nil
	}
	var paths []string
	var walk func(v any, path string)
	walk = func(v any, path string) {
		switch v := v.(type) {
		case map[string]any:
			if _, has := v[field]; !has && hasKeys(v, owner) {
				paths = append(paths, path)
			}
			keys := make([]string, 0, len(v))
			for k := range v {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				walk(v[k], joinPath(path, k))
			}
		case []any:
			for i, item := range v {
				walk(item, fmt.Sprintf("%s[%d]", path, i))
			}
		}
	}
	walk(def, "")
	return paths
}

func hasKeys(m map[string]any, keys []string) bool {
	for _, k := range keys {
		if _, ok := m[k]; !ok {
			return false
		}
	}
	return true
}

func joinPath(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

// describePaths lists up to maxPaths paths for a message.
func describePaths(paths []string) string {
	if len(paths) <= maxPaths {
		return strings.Join(paths, ", ")
	}
	return fmt.Sprintf("%s and %d more", strings.Join(paths[:maxPaths], ", "), len(paths)-maxPaths)
}
