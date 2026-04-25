package translation

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/cli/mgmnt"
)

type TranslationWriteStats struct {
	Total   int
	Success int
	Failed  int
	Errors  []string
}

type TranslationPushStats struct {
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
	defaultDir := filepath.Join(".", "suprsend", "translations")
	fmt.Fprintf(os.Stdout, "Where would you like to save the translations?\n")
	fmt.Fprintf(os.Stdout, "Default: %s\n", defaultDir)
	fmt.Fprintf(os.Stdout, "Enter directory path (or press Enter for default): ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultDir, true
	}
	return input, true
}

func WriteTranslationToFiles(resp mgmnt.TranslationResponse, outputDir string) (*TranslationWriteStats, error) {
	stats := &TranslationWriteStats{
		Total:  len(resp.Results),
		Errors: []string{},
	}

	info, err := os.Stat(outputDir)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				return stats, err
			}
		} else {
			errMsg := fmt.Sprintf("error accessing '%s': %v", outputDir, err)
			return stats, fmt.Errorf(errMsg)
		}
	} else if !info.IsDir() {
		return stats, err
	}

	for _, wf := range resp.Results {
		obj, ok := wf.(map[string]any)
		if !ok {
			stats.Failed++
			stats.Errors = append(stats.Errors, "Invalid  format")
			continue
		}
		slug, _ := obj["slug"].(string)
		filename := filepath.Join(outputDir, obj["filename"].(string))
		content, ok := obj["content"]
		if !ok || content == nil {
			log.Warnf("No content found for translation '%s', skipping", slug)
			stats.Failed++
			stats.Errors = append(stats.Errors, fmt.Sprintf("No content found for translation '%s'", slug))
			continue
		}
		fileData, err := json.MarshalIndent(content, "", "  ")
		if err != nil {
			log.WithError(err).Errorf("Failed to marshal translation '%s'", slug)
			stats.Failed++
			stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to marshal translation '%s': %v", slug, err))
			continue
		}
		if err := os.WriteFile(filename, fileData, 0644); err != nil {
			log.WithError(err).Errorf("Failed to write file '%s'", filename)
			stats.Failed++
			stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to write file '%s': %v", filename, err))
			continue
		}
		log.Infof("Wrote translation to %s", filename)
		stats.Success++
	}
	return stats, nil
}
