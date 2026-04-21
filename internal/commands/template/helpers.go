package template

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
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

func WriteTemplatesToFiles(results []TemplateResult, outputDir string) (*TemplateWriteStats, error) {
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
			debugErrorLog("Error: %s", err)
			stats.Failed++
			stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to marshal template.json for '%s': %v", tmpl.Slug, err))
			continue
		}
		templateFile := filepath.Join(templateDir, "template.json")
		if err := os.WriteFile(templateFile, templateJSON, 0o644); err != nil {
			debugErrorLog("Error: %s", err)
			stats.Failed++
			stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to write template.json for '%s': %v", tmpl.Slug, err))
			continue
		}
		debugLog("Wrote: %s", templateFile)
		fmt.Fprintf(os.Stdout, "Wrote template metadata to %s\n", templateFile)

		// Write mock_data.json if present
		if tmpl.MockData != nil {
			mockJSON, err := json.MarshalIndent(tmpl.MockData, "", "  ")
			if err != nil {
				debugErrorLog("Error marshaling mock_data.json for '%s': %s", tmpl.Slug, err)
			} else {
				mockFile := filepath.Join(templateDir, "mock_data.json")
				if err := os.WriteFile(mockFile, mockJSON, 0o644); err != nil {
					debugErrorLog("Error writing mock_data.json for '%s': %s", tmpl.Slug, err)
				} else {
					debugLog("Wrote: %s", mockFile)
					fmt.Fprintf(os.Stdout, "Wrote mock data to %s\n", mockFile)
				}
			}
		}

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
	}

	return stats, nil
}

