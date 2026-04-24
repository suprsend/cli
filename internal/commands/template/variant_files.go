package template

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/suprsend/cli/internal/utils"
)

// Variant represents a single template variant as an arbitrary JSON object.
type Variant = map[string]any

// fileRefConfig defines how a variant field is extracted to / reassembled from a separate file.
type fileRefConfig struct {
	ShouldExtract func(variant Variant) bool // if non-nil and returns false, skip extraction for this variant
	Filename      func(variant Variant) string
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
func shouldExtractMsTeamsCard(variant Variant) bool {
	ch, _ := variant["channel"].(string)
	if ch != "ms_teams" {
		return false
	}
	_, _, lang, ok := getNestedValue(variant, "content.templating_language")
	if !ok {
		return false
	}
	s, _ := lang.(string)
	return s == "jsonnet"
}
func shouldExtractInbox(variant Variant) bool {
	ch, _ := variant["channel"].(string)
	return ch == "inbox"
}

// fileRefKeys defines which variant keys should be extracted into separate files.
// Map key: dot-notation path, value: config with filename and whether to keep a @ref or delete the key.
// Add new entries here to extract more fields.
var fileRefKeys = map[string]fileRefConfig{
	"content.body.designer.design_json": {Filename: func(_ Variant) string { return "body.designer.json" }, ShouldExtract: shouldExtractDesigner},
	"content.body.designer.html":        {Filename: func(_ Variant) string { return "body.designer.html" }, ShouldExtract: shouldExtractDesigner},
	"content.body.designer.text":        {Filename: func(_ Variant) string { return "body.designer.txt" }, ShouldExtract: shouldExtractDesigner},
	"content.body.raw.html":             {Filename: func(_ Variant) string { return "body.raw.html" }, ShouldExtract: shouldExtractRaw},
	"content.body.raw.text":             {Filename: func(_ Variant) string { return "body.raw.txt" }, ShouldExtract: shouldExtractRaw},
	"content.body.plain_text.text":      {Filename: func(_ Variant) string { return "body.plain_text.txt" }, ShouldExtract: shouldExtractPlainText},
	"content.body_block":                {Filename: func(_ Variant) string { return "body.block.jsonnet" }, StringifyJSON: true, ShouldExtract: shouldExtractSlackBlock},
	"content.body_card":                 {Filename: func(_ Variant) string { return "body.card.jsonnet" }, StringifyJSON: true, ShouldExtract: shouldExtractMsTeamsCard},
	"content.body":                      {Filename: func(_ Variant) string { return "body.md" }, ShouldExtract: shouldExtractInbox},
	// "content.body_text":                 {Filename: func(_ Variant) string { return "body_text.txt" }},
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
	filesToWrite := map[string]string{} // filename -> content

	for _, path := range sortedFileRefPaths(true) {
		cfg := fileRefKeys[path]
		parent, lastKey, val, ok := getNestedValue(variantCopy, path)

		if cfg.ShouldExtract != nil && !cfg.ShouldExtract(variantCopy) {
			continue
		}

		if !ok {
			continue
		}

		content, ok := valueToFileContent(val)
		if !ok {
			continue
		}
		filename := cfg.Filename(variantCopy)
		filesToWrite[filename] = content
		parent[lastKey] = "@" + filename
	}

	// Marshal variant JSON and include it alongside the extracted content files
	variantJSON, err := json.MarshalIndent(variantCopy, "", "  ")
	if err != nil {
		log.Errorf("Failed to marshal variant '%s/%s' for '%s': %v", channel, variantName, slug, err)
		stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to marshal variant '%s/%s' for '%s': %v", channel, variantName, slug, err))
		return err
	}
	filesToWrite["variant.json"] = string(variantJSON)

	for filename, content := range filesToWrite {
		filePath := filepath.Join(variantDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
			log.Errorf("Failed to write '%s': %v", filePath, err)
			stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to write '%s': %v", filePath, err))
			return err
		}
		log.Debugf("Wrote: %s", filePath)
	}
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
			// Non-string value: for JSON-backed fields, accept structured data as-is
			if filepath.Ext(cfg.Filename(variant)) == ".json" {
				switch val.(type) {
				case map[string]any, []any:
					// valid JSON structure, keep it
				default:
					log.Warnf("Inline value at %s has unsupported type %T", path, val)
				}
			}
			continue
		}

		filename := cfg.Filename(variant)
		if !isFileRef(strVal, filename) {
			continue
		}

		if _, statErr := os.Stat(filepath.Join(variantDir, filename)); statErr != nil {
			log.Warnf("Referenced file %s not found at %s: %v", filename, path, statErr)
			continue
		}
		content, readErr := readExtractedFile(variantDir, filename, filepath.Ext(filename))
		if readErr != nil {
			log.Warnf("Failed to read extracted file %s: %v", filename, readErr)
			continue
		}

		if cfg.StringifyJSON {
			switch content.(type) {
			case map[string]any, []any:
				b, err := json.Marshal(content)
				if err != nil {
					log.Warnf("Failed to stringify JSON at %s: %v", path, err)
					continue
				}
				content = string(b)
			}
		}

		parent[lastKey] = content
	}

	return variant, nil
}

// isFileRef checks whether a string value is an @file reference
// pointing at the filename this field is configured to extract to.
func isFileRef(val, expectedFilename string) bool {
	return val == "@"+expectedFilename
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
