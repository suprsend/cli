package template

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// fileRefKeys defines which variant keys should be extracted into separate files.
// Map key: dot-notation path, value: config with file extension and whether to keep a @ref or delete the key.
// Add new entries here to extract more fields.
type fileRefConfig struct {
	Ext     string
	KeepRef bool // true: replace with @filename, false: delete the key from parent
}

var fileRefKeys = map[string]fileRefConfig{
	"content":                           {Ext: ".json", KeepRef: false},
	"content.body.designer.design_json": {Ext: ".json", KeepRef: true},
	"content.body.designer.html":        {Ext: ".html", KeepRef: true},
	"content.body.designer.text":        {Ext: ".txt", KeepRef: true},
	"content.body.raw.html":             {Ext: ".html", KeepRef: true},
	"content.body.raw.text":             {Ext: ".txt", KeepRef: true},
	"content.body_text":                 {Ext: ".txt", KeepRef: true},
}

type TemplateWriteStats struct {
	Total   int
	Success int
	Failed  int
	Errors  []string
}

func isDebugMode() bool {
	return viper.GetBool("debug")
}

func debugLog(format string, args ...any) {
	if isDebugMode() {
		log.Infof(format, args...)
	}
}

func debugErrorLog(format string, args ...any) {
	if isDebugMode() {
		log.Errorf(format, args...)
	}
}

func promptForOutputDirectory() string {
	reader := bufio.NewReader(os.Stdin)
	defaultDir := filepath.Join(".", "suprsend", "templates")
	fmt.Fprintf(os.Stdout, "Where would you like to save the templates?\n")
	fmt.Fprintf(os.Stdout, "Default: %s\n", defaultDir)
	fmt.Fprintf(os.Stdout, "Enter directory path (or press Enter for default): ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultDir
	}
	return input
}

func ensureOutputDirectory(dirPath string) error {
	info, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stdout, "Creating directory: %s\n", dirPath)
			return os.MkdirAll(dirPath, 0o755)
		}
		return fmt.Errorf("error checking directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path '%s' exists but is not a directory", dirPath)
	}
	if info.Mode().Perm()&0o200 == 0 {
		return fmt.Errorf("directory '%s' is not writable", dirPath)
	}
	return nil
}

func WriteTemplatesToFiles(results []templateResult, outputDir string) (*TemplateWriteStats, error) {
	stats := &TemplateWriteStats{
		Total:  len(results),
		Errors: []string{},
	}

	info, err := os.Stat(outputDir)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(outputDir, 0o755); err != nil {
				return stats, err
			}
		} else {
			return stats, fmt.Errorf("error accessing '%s': %v", outputDir, err)
		}
	} else if !info.IsDir() {
		return stats, fmt.Errorf("path '%s' exists but is not a directory", outputDir)
	}

	for _, tmpl := range results {
		templateDir := filepath.Join(outputDir, tmpl.Slug)
		if err := os.MkdirAll(templateDir, 0o755); err != nil {
			stats.Failed++
			stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to create directory for template '%s': %v", tmpl.Slug, err))
			continue
		}

		// Write main.json with template-level metadata
		mainData := map[string]any{
			"slug":             tmpl.Slug,
			"name":             tmpl.Name,
			"enabled_channels": tmpl.EnabledChannels,
		}
		mainJSON, err := json.MarshalIndent(mainData, "", "  ")
		if err != nil {
			debugErrorLog("Error: %s", err)
			stats.Failed++
			stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to marshal main.json for '%s': %v", tmpl.Slug, err))
			continue
		}
		mainFile := filepath.Join(templateDir, "main.json")
		if err := os.WriteFile(mainFile, mainJSON, 0o644); err != nil {
			debugErrorLog("Error: %s", err)
			stats.Failed++
			stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to write main.json for '%s': %v", tmpl.Slug, err))
			continue
		}
		debugLog("Wrote: %s", mainFile)
		fmt.Fprintf(os.Stdout, "Wrote template metadata to %s\n", mainFile)

		// Write each variant
		for _, variant := range tmpl.Variants {
			channel, _ := variant["channel"].(string)
			variantName, _ := variant["id"].(string)

			if channel == "" || variantName == "" {
				debugErrorLog("Skipping variant with missing channel or id for template '%s'", tmpl.Slug)
				continue
			}

			var variantDir string
			tenantID, _ := variant["tenant_id"].(string)
			if tenantID != "" {
				variantDir = filepath.Join(templateDir, channel, "__tenant_overrides__", tenantID, variantName)
			} else {
				variantDir = filepath.Join(templateDir, channel, variantName)
			}
			if err := os.MkdirAll(variantDir, 0o755); err != nil {
				stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to create variant directory '%s': %v", variantDir, err))
				continue
			}

			if err := writeVariantFiles(variantDir, variant, channel, variantName, tmpl.Slug, stats); err != nil {
				continue
			}
		}

		stats.Success++
	}

	return stats, nil
}

func deepCopyMap(m map[string]any) map[string]any {
	b, _ := json.Marshal(m)
	var out map[string]any
	json.Unmarshal(b, &out)
	return out
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

func writeVariantFiles(variantDir string, variant map[string]any, channel, variantName, slug string, stats *TemplateWriteStats) error {
	variantCopy := deepCopyMap(variant)
	extractedFiles := map[string]string{} // filename -> content

	// Sort keys deepest-first so nested extractions happen before their parents
	sortedPaths := make([]string, 0, len(fileRefKeys))
	for path := range fileRefKeys {
		sortedPaths = append(sortedPaths, path)
	}
	sort.Slice(sortedPaths, func(i, j int) bool {
		return strings.Count(sortedPaths[i], ".") > strings.Count(sortedPaths[j], ".")
	})

	for _, path := range sortedPaths {
		cfg := fileRefKeys[path]
		parent, lastKey, val, ok := getNestedValue(variantCopy, path)
		if !ok || val == nil {
			continue
		}

		content, hasContent := valueToFileContent(val)
		if !hasContent {
			continue
		}

		filename := strings.ReplaceAll(path, ".", "_") + cfg.Ext
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
