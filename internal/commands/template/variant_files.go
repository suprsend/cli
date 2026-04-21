package template

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/suprsend/cli/internal/utils"
)

// fileRefConfig defines how a variant field is extracted to / reassembled from a separate file.
type fileRefConfig struct {
	Filename    string
	KeepRef     bool                              // true: replace with @filename, false: delete the key from parent
	Inline      bool                              // true: accept literal values (not just @file refs) during reassembly
	ForceCreate func(variant map[string]any) bool // if non-nil and returns true, always write the file even when the value is empty
}

// variantIs returns true when the variant matches the given channel and the value at bodyTypePath equals bodyType.
func variantIs(variant map[string]any, channel, bodyTypePath, bodyType string) bool {
	if ch, _ := variant["channel"].(string); ch != channel {
		return false
	}
	_, _, bt, ok := getNestedValue(variant, bodyTypePath)
	if !ok {
		return false
	}
	t, _ := bt.(string)
	return t == bodyType
}

func forceCreateRaw(variant map[string]any) bool {
	return variantIs(variant, "email", "content.body.type", "raw")
}
func forceCreateDesigner(variant map[string]any) bool {
	return variantIs(variant, "email", "content.body.type", "designer")
}
func forceCreatePlainText(variant map[string]any) bool {
	return variantIs(variant, "email", "content.body.type", "plain_text")
}
func forceCreateSlackBlock(variant map[string]any) bool {
	return variantIs(variant, "slack", "content.body_type", "block")
}

// fileRefKeys defines which variant keys should be extracted into separate files.
// Map key: dot-notation path, value: config with filename and whether to keep a @ref or delete the key.
// Add new entries here to extract more fields.
var fileRefKeys = map[string]fileRefConfig{
	"content.body.designer.design_json": {Filename: "design.json", KeepRef: true, Inline: true, ForceCreate: forceCreateDesigner},
	"content.body.designer.html":        {Filename: "body.designer.html", KeepRef: true, Inline: true, ForceCreate: forceCreateDesigner},
	"content.body.designer.text":        {Filename: "body.designer.txt", KeepRef: true, Inline: true, ForceCreate: forceCreateDesigner},
	"content.body.raw.html":             {Filename: "body.raw.html", KeepRef: true, Inline: true, ForceCreate: forceCreateRaw},
	"content.body.raw.text":             {Filename: "body.raw.txt", KeepRef: true, Inline: true, ForceCreate: forceCreateRaw},
	"content.body.plain_text.text":      {Filename: "body.plain_text.txt", KeepRef: true, Inline: true, ForceCreate: forceCreatePlainText},
	"content.body_block":                {Filename: "body.block.json", KeepRef: true, Inline: true, ForceCreate: forceCreateSlackBlock},
	// "content.body_text":                 {Filename: "body_text.txt", KeepRef: true, Inline: true},
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
func getNestedValue(m map[string]any, path string) (parent map[string]any, lastKey string, val any, ok bool) {
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

// setNestedValue sets a value at a dot-notation path, creating intermediate maps as needed.
func setNestedValue(m map[string]any, path string, val any) {
	parts := strings.Split(path, ".")
	current := m
	for i, p := range parts {
		if i == len(parts)-1 {
			current[p] = val
			return
		}
		next, exists := current[p]
		if !exists {
			newMap := map[string]any{}
			current[p] = newMap
			current = newMap
			continue
		}
		nextMap, isMap := next.(map[string]any)
		if !isMap {
			newMap := map[string]any{}
			current[p] = newMap
			current = newMap
			continue
		}
		current = nextMap
	}
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

func writeVariantFiles(variantDir string, variant map[string]any, channel, variantName, slug string, stats *TemplateWriteStats) error {
	variantCopy := utils.DeepCopyMap(variant)
	extractedFiles := map[string]string{} // filename -> content

	for _, path := range sortedFileRefPaths(true) {
		cfg := fileRefKeys[path]
		parent, lastKey, val, ok := getNestedValue(variantCopy, path)

		forced := cfg.ForceCreate != nil && cfg.ForceCreate(variantCopy)

		if cfg.ForceCreate != nil && !forced {
			filePath := filepath.Join(variantDir, cfg.Filename)
			if _, statErr := os.Stat(filePath); statErr == nil {
				fmt.Fprintf(os.Stdout, "  Warning: %s exists but is being ignored (variant type mismatch — file not applicable for this variant)\n", filePath)
			}
		}

		if !ok || val == nil {
			if forced {
				extractedFiles[cfg.Filename] = ""
				setNestedValue(variantCopy, path, "@"+cfg.Filename)
			}
			continue
		}

		content, hasContent := valueToFileContent(val)
		if !hasContent {
			if forced {
				extractedFiles[cfg.Filename] = ""
				if cfg.KeepRef {
					parent[lastKey] = "@" + cfg.Filename
				} else {
					delete(parent, lastKey)
				}
			}
			continue
		}

		filename := cfg.Filename
		extractedFiles[filename] = content
		if cfg.KeepRef {
			parent[lastKey] = "@" + filename
		} else {
			delete(parent, lastKey)
		}
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
		fmt.Fprintf(os.Stdout, "  Wrote %s\n", filePath)
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
	fmt.Fprintf(os.Stdout, "  Wrote variant to %s\n", variantFile)
	return nil
}

// --- Join: reassemble variant from separate files (used by push) ---

// readAndAssembleVariant reads a variant.json and re-inlines any extracted @file references.
func readAndAssembleVariant(variantDir string) (map[string]any, error) {
	variantFile := filepath.Join(variantDir, "variant.json")
	data, err := os.ReadFile(variantFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read variant.json: %w", err)
	}

	var variant map[string]any
	if err := json.Unmarshal(data, &variant); err != nil {
		return nil, fmt.Errorf("failed to parse variant.json: %w", err)
	}

	// Process paths shallowest-first so parent keys exist when we set nested values
	for _, path := range sortedFileRefPaths(false) {
		cfg := fileRefKeys[path]
		parent, lastKey, val, ok := getNestedValue(variant, path)

		if !ok {
			// For keys with KeepRef=false (like "content"), the key was deleted from variant.json.
			// Check if the extracted file exists and re-inline it.
			if !cfg.KeepRef {
				filename := cfg.Filename
				content, readErr := readExtractedFile(variantDir, filename, filepath.Ext(cfg.Filename))
				if readErr == nil {
					setNestedValue(variant, path, content)
				}
			}
			//todo: may be we need to log missing file for non-keepRef keys?
			continue
		}

		// Check if the value is a @file reference
		strVal, isStr := val.(string)
		if !isStr {
			// Non-string value: if inline is enabled and ext is JSON, accept structured data as-is
			if cfg.Inline && filepath.Ext(cfg.Filename) == ".json" {
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
				if filepath.Ext(cfg.Filename) == ".json" {
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

		parent[lastKey] = content
	}

	return variant, nil
}

// fileRefPattern matches valid extracted file references: word chars, dots, hyphens, with a known extension.
var fileRefPattern = regexp.MustCompile(`^[\w][\w.\-]*\.(json|html|txt)$`)

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
