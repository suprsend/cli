package template

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/utils"
)

type TemplateWriteStats struct {
	Total   int
	Success int
	Failed  int
	Errors  []string
}

type TemplatePushStats struct {
	Total   int
	Success int
	Failed  int
	Errors  []string
}

func promptForOutputDirectory() (string, bool) {
	if !utils.IsInputInteractive() {
		fmt.Fprintf(os.Stderr, "required flag missing, cannot prompt in non-interactive mode")
		return "", false
	}
	reader := bufio.NewReader(os.Stdin)
	defaultDir := filepath.Join(".", "suprsend", "templates")
	fmt.Fprintf(os.Stdout, "Where would you like to save the templates?\n")
	fmt.Fprintf(os.Stdout, "Default: %s\n", defaultDir)
	fmt.Fprintf(os.Stdout, "Enter directory path (or press Enter for default): ")
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v. Using default directory: %s\n", err, defaultDir)
		return defaultDir, true
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultDir, true
	}
	return input, true
}

func ensureOutputDirectory(dirPath string) error {
	info, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Infof("Creating directory: %s", dirPath)
			if err := os.MkdirAll(dirPath, 0o755); err != nil {
				return clierr.Wrap(err, clierr.CodeFileNotFound, fmt.Sprintf("failed to create directory '%s'", dirPath))
			}
			return nil
		}
		return clierr.Wrap(err, clierr.CodeFileNotFound, fmt.Sprintf("error checking directory '%s'", dirPath))
	}
	if !info.IsDir() {
		return clierr.New(fmt.Sprintf("path '%s' exists but is not a directory", dirPath), clierr.CodeInvalidUsage)
	}
	if info.Mode().Perm()&0o200 == 0 {
		return clierr.New(fmt.Sprintf("directory '%s' is not writable", dirPath), clierr.CodeInvalidUsage)
	}
	return nil
}

// EnsureTemplatesOutputDir verifies (or creates) the top-level output
// directory. Streaming callers invoke this once before the per-template
// callback fires, since each WriteOneTemplate assumes the parent exists.
func EnsureTemplatesOutputDir(outputDir string) error {
	info, err := os.Stat(outputDir)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(outputDir, 0o755); err != nil {
				return clierr.Wrap(err, clierr.CodeFileNotFound, fmt.Sprintf("failed to create output directory '%s'", outputDir))
			}
			return nil
		}
		return clierr.Wrap(err, clierr.CodeFileNotFound, fmt.Sprintf("error accessing '%s'", outputDir))
	}
	if !info.IsDir() {
		return clierr.New(fmt.Sprintf("path '%s' exists but is not a directory", outputDir), clierr.CodeInvalidUsage)
	}
	return nil
}

// WriteOneTemplate writes a single template (template.json + mock_data.json
// + per-variant directories) into outputDir/<slug>/. The existing per-slug
// directory is removed first so renames / deletions of variants don't leave
// stale files behind. stats is mutated in place; callers that share stats
// across goroutines must serialize access.
func WriteOneTemplate(tmpl TemplateResult, outputDir string, stats *TemplateWriteStats) error {
	templateDir := filepath.Join(outputDir, tmpl.Slug)
	if err := os.RemoveAll(templateDir); err != nil {
		return clierr.Wrap(err, clierr.CodeAPIInternal, fmt.Sprintf("failed to remove existing directory for template '%s'", tmpl.Slug))
	}
	return writeOneTemplate(tmpl, outputDir, stats)
}

func WriteTemplatesToFiles(results []TemplateResult, outputDir string) (*TemplateWriteStats, error) {
	stats := &TemplateWriteStats{
		Total:  len(results),
		Errors: []string{},
	}

	if err := EnsureTemplatesOutputDir(outputDir); err != nil {
		return stats, err
	}

	for _, tmpl := range results {
		templateDir := filepath.Join(outputDir, tmpl.Slug)
		if err := os.RemoveAll(templateDir); err != nil {
			return stats, clierr.Wrap(err, clierr.CodeAPIInternal, fmt.Sprintf("failed to remove existing directory for template '%s'", tmpl.Slug))
		}
	}

	for _, tmpl := range results {
		_ = writeOneTemplate(tmpl, outputDir, stats)
	}
	return stats, nil
}

// writeOneTemplate is the shared body — assumes outputDir exists and the
// per-slug directory has already been removed.
func writeOneTemplate(tmpl TemplateResult, outputDir string, stats *TemplateWriteStats) error {
	templateDir := filepath.Join(outputDir, tmpl.Slug)
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		stats.Failed++
		stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to create directory for template '%s': %v", tmpl.Slug, err))
		return nil
	}

	// Write template.json with template-level metadata
	templateData := map[string]any{
		"slug":             tmpl.Slug,
		"name":             tmpl.Name,
		"enabled_channels": tmpl.EnabledChannels,
	}
	if tmpl.VariantOrder != nil {
		flatOrder := map[string][]string{}
		for _, ch := range tmpl.VariantOrder.Channels {
			for _, tenant := range ch.Tenants {
				key := ch.Channel
				if tenant.TenantID != nil {
					key = ch.Channel + "/" + *tenant.TenantID
				}
				flatOrder[key] = tenant.Variants
			}
		}
		templateData["variant_order"] = flatOrder
	}
	templateJSON, err := json.MarshalIndent(templateData, "", "  ")
	if err != nil {
		log.Errorf("Failed to marshal template.json for '%s': %v", tmpl.Slug, err)
		stats.Failed++
		stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to marshal template.json for '%s': %v", tmpl.Slug, err))
		return nil
	}
	templateFile := filepath.Join(templateDir, "template.json")
	if err := os.WriteFile(templateFile, templateJSON, 0o644); err != nil {
		log.Errorf("Failed to write template.json for '%s': %v", tmpl.Slug, err)
		stats.Failed++
		stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to write template.json for '%s': %v", tmpl.Slug, err))
		return nil
	}
	log.Debugf("Wrote: %s", templateFile)

	// Write mock_data.json if present
	if tmpl.MockData != nil {
		mockJSON, err := json.MarshalIndent(tmpl.MockData, "", "  ")
		if err != nil {
			log.Errorf("Error marshaling mock_data.json for '%s': %s", tmpl.Slug, err)
		} else {
			mockFile := filepath.Join(templateDir, "mock_data.json")
			if err := os.WriteFile(mockFile, mockJSON, 0o644); err != nil {
				log.Errorf("Error writing mock_data.json for '%s': %s", tmpl.Slug, err)
			} else {
				log.Debugf("Wrote: %s", mockFile)
			}
		}
	}

	// Write each variant
	for _, variant := range tmpl.Variants {
		channel, _ := variant["channel"].(string)
		variantName, _ := variant["id"].(string)

		if channel == "" || variantName == "" {
			log.Errorf("Skipping variant with missing channel or id for template '%s'", tmpl.Slug)
			continue
		}

		var variantDir string
		tenantID, _ := variant["tenant_id"].(string)
		if tenantID != "" {
			variantDir = filepath.Join(templateDir, channel, "_tenants", tenantID, variantName)
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
	log.Infof("Pulled template: %s", tmpl.Slug)
	return nil
}
