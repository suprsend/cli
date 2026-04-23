package template

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/suprsend/cli/internal/utils"
)

// Variant represents a single template variant as an arbitrary JSON object.
type Variant = map[string]any

// fileRefConfig defines how a variant field is extracted to / reassembled from a separate file.
type fileRefConfig struct {
	ShouldExtract func(variant Variant) bool // if non-nil and returns true, always write the file even when the value is empty
	Filename      func(variant Variant) string
	Inline        bool // true: accept literal values (not just @file refs) during reassembly
	StringifyJSON bool // true: re-serialize parsed JSON back to a string on reassembly (for API fields that expect a JSON string, not an object)
}

// variantIs returns true when the variant matches one of the given channels and the value at bodyTypePath equals bodyType.
func variantIs(variant Variant, channels []string, bodyTypePath, bodyType string) bool {
	ch, _ := variant["channel"].(string)
	if !slices.Contains(channels, ch) {
		return false
	}
	_, _, bt, ok := getNestedValue(variant, bodyTypePath)
	if !ok {
		return false
	}
	t, _ := bt.(string)
	return t == bodyType
}

func shouldExtractRaw(variant Variant) bool {
	return variantIs(variant, []string{"email"}, "content.body.type", "raw")
}
func shouldExtractDesigner(variant Variant) bool {
	return variantIs(variant, []string{"email"}, "content.body.type", "designer")
}
func shouldExtractPlainText(variant Variant) bool {
	return variantIs(variant, []string{"email"}, "content.body.type", "plain_text")
}
func shouldExtractSlackBlock(variant Variant) bool {
	return variantIs(variant, []string{"slack", "ms_teams"}, "content.body_type", "block")
}
func shouldExtractInbox(variant Variant) bool {
	ch, _ := variant["channel"].(string)
	return ch == "inbox"
}

// fileRefKeys defines which variant keys should be extracted into separate files.
// Map key: dot-notation path, value: config with filename and whether to keep a @ref or delete the key.
// Add new entries here to extract more fields.
var fileRefKeys = map[string]fileRefConfig{
	"content.body.designer.design_json": {Filename: func(_ Variant) string { return "body.designer.json" }, Inline: true, ShouldExtract: shouldExtractDesigner},
	"content.body.designer.html":        {Filename: func(_ Variant) string { return "body.designer.html" }, Inline: true, ShouldExtract: shouldExtractDesigner},
	"content.body.designer.text":        {Filename: func(_ Variant) string { return "body.designer.txt" }, Inline: true, ShouldExtract: shouldExtractDesigner},
	"content.body.raw.html":             {Filename: func(_ Variant) string { return "body.raw.html" }, Inline: true, ShouldExtract: shouldExtractRaw},
	"content.body.raw.text":             {Filename: func(_ Variant) string { return "body.raw.txt" }, Inline: true, ShouldExtract: shouldExtractRaw},
	"content.body.plain_text.text":      {Filename: func(_ Variant) string { return "body.plain_text.txt" }, Inline: true, ShouldExtract: shouldExtractPlainText},
	"content.body_block":                {Filename: func(_ Variant) string { return "body.block.jsonnet" }, Inline: true, StringifyJSON: true, ShouldExtract: shouldExtractSlackBlock},
	"content.body":                      {Filename: func(_ Variant) string { return "body.md" }, Inline: true, ShouldExtract: shouldExtractInbox},
	// "content.body_text":                 {Filename: func(_ Variant) string { return "body_text.txt" }, Inline: true},
}

// sortedFileRefPaths returns fileRefKeys paths sorted by depth.
// descending=true: deepest first (for extraction/split).
// descending=false: shallowest first (for reassembly/join).
func sortedFileRefPaths(descending bool) []string {
	paths := make([]string, 0, len(fileRefKeys))
	for path := range fileRefKeys {
		paths = append(paths, path)
	}
	sort.Slice(paths, func(i, j int) bool {
		di := strings.Count(paths[i], ".")
		dj := strings.Count(paths[j], ".")
		if descending {
			return di > dj
		}
		return di < dj
	})
	return paths
}

// getNestedValue walks a dot-notation path and returns the parent map, the final key, and the value.
func getNestedValue(m Variant, path string) (parent Variant, lastKey string, val any, ok bool) {
	parts := strings.Split(path, ".")
	current := m
	for i, p := range parts {
		if i == len(parts)-1 {
			v, exists := current[p]
			return current, p, v, exists
		}
		next, exists := current[p]
		if !exists {
			return nil, "", nil, false
		}
		nextMap, isMap := next.(map[string]any)
		if !isMap {
			return nil, "", nil, false
		}
		current = nextMap
	}
	return nil, "", nil, false
}

// valueToFileContent converts a value to a string suitable for writing to a file.
func valueToFileContent(v any) (string, bool) {
	switch val := v.(type) {
	case string:
		if val == "" {
			return "", false
		}
		return val, true
	case map[string]any, []any:
		b, err := json.MarshalIndent(val, "", "  ")
		if err != nil {
			return "", false
		}
		return string(b), true
	default:
		return "", false
	}
}

// --- Split: extract variant fields into separate files (used by pull) ---

func writeVariantFiles(variantDir string, variant Variant, channel, variantName, slug string, stats *TemplateWriteStats) error {
	variantCopy := utils.DeepCopyMap(variant)
	extractedFiles := map[string]string{} // filename -> content

	for _, path := range sortedFileRefPaths(true) {
		cfg := fileRefKeys[path]
		parent, lastKey, val, ok := getNestedValue(variantCopy, path)

		if cfg.ShouldExtract != nil && !cfg.ShouldExtract(variantCopy) {
			continue
		}

		if !ok {
			continue
		}

		content, _ := valueToFileContent(val)
		filename := cfg.Filename(variantCopy)
		extractedFiles[filename] = content
		parent[lastKey] = "@" + filename
	}

	// Write extracted content files
	for filename, content := range extractedFiles {
		filePath := filepath.Join(variantDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
			debugErrorLog("Error: %s", err)
			stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to write '%s': %v", filePath, err))
			return err
		}
		debugLog("Wrote: %s", filePath)
	}

	// Marshal variant JSON
	variantJSON, err := json.MarshalIndent(variantCopy, "", "  ")
	if err != nil {
		debugErrorLog("Error: %s", err)
		stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to marshal variant '%s/%s' for '%s': %v", channel, variantName, slug, err))
		return err
	}

	variantFile := filepath.Join(variantDir, "variant.json")
	if err := os.WriteFile(variantFile, variantJSON, 0o644); err != nil {
		debugErrorLog("Error: %s", err)
		stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to write variant file '%s': %v", variantFile, err))
		return err
	}
	debugLog("Wrote: %s", variantFile)
	return nil
}

// --- Join: reassemble variant from separate files (used by push) ---

// readAndAssembleVariant reads a variant.json and re-inlines any extracted @file references.
func readAndAssembleVariant(variantDir string) (Variant, error) {
	variantFile := filepath.Join(variantDir, "variant.json")
	data, err := os.ReadFile(variantFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read variant.json: %w", err)
	}

	var variant Variant
	if err := json.Unmarshal(data, &variant); err != nil {
		return nil, fmt.Errorf("failed to parse variant.json: %w", err)
	}

	// Process paths shallowest-first so parent keys exist when we set nested values
	for _, path := range sortedFileRefPaths(false) {
		cfg := fileRefKeys[path]
		parent, lastKey, val, ok := getNestedValue(variant, path)

		if !ok {
			continue
		}

		// Check if the value is a @file reference
		strVal, isStr := val.(string)
		if !isStr {
			// Non-string value: if inline is enabled and ext is JSON, accept structured data as-is
			if cfg.Inline && filepath.Ext(cfg.Filename(variant)) == ".json" {
				switch val.(type) {
				case map[string]any, []any:
					// valid JSON structure, keep it
				default:
					debugErrorLog("Inline value at %s has unsupported type %T", path, val)
				}
			}
			continue
		}

		if !isFileRef(variantDir, strVal) {
			// Not a valid file ref: if inline is enabled, accept the literal value
			if cfg.Inline {
				if filepath.Ext(cfg.Filename(variant)) == ".json" {
					var jsonVal any
					if err := json.Unmarshal([]byte(strVal), &jsonVal); err != nil {
						debugErrorLog("Inline JSON value at %s is not valid JSON: %v", path, err)
						continue
					}
					parent[lastKey] = jsonVal
				}
				// For .html/.txt, the string value is already correct
			}
			continue
		}

		filename := strings.TrimPrefix(strVal, "@")
		content, readErr := readExtractedFile(variantDir, filename, filepath.Ext(filename))
		if readErr != nil {
			debugErrorLog("Failed to read extracted file %s: %v", filename, readErr)
			continue
		}

		if s, ok := content.(string); ok && s == "" {
			parent[lastKey] = nil
			continue
		}

		if cfg.StringifyJSON {
			switch content.(type) {
			case map[string]any, []any:
				b, err := json.Marshal(content)
				if err != nil {
					debugErrorLog("Failed to stringify JSON at %s: %v", path, err)
					continue
				}
				content = string(b)
			}
		}

		parent[lastKey] = content
	}

	return variant, nil
}

// fileRefPattern matches valid extracted file references: word chars, dots, hyphens, with a known extension.
var fileRefPattern = regexp.MustCompile(`^[\w][\w.\-]*\.(json|html|txt|jsonnet|md)$`)

// isFileRef checks whether a string value is a valid @file reference.
// It verifies the @-prefix, the filename matches the expected pattern, and the file exists on disk.
func isFileRef(variantDir, val string) bool {
	if !strings.HasPrefix(val, "@") {
		return false
	}
	filename := strings.TrimPrefix(val, "@")
	if !fileRefPattern.MatchString(filename) {
		return false
	}
	_, err := os.Stat(filepath.Join(variantDir, filename))
	return err == nil
}

// readExtractedFile reads an extracted file and returns the appropriate typed value.
func readExtractedFile(variantDir, filename, ext string) (any, error) {
	filePath := filepath.Join(variantDir, filename)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// For JSON files, parse back into structured data
	if ext == ".json" {
		var jsonVal any
		if err := json.Unmarshal(data, &jsonVal); err != nil {
			return string(data), nil // fall back to string if not valid JSON
		}
		return jsonVal, nil
	}

	return string(data), nil
}
