package workflow

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

type WorkflowWriteStats struct {
	Total   int
	Success int
	Failed  int
	Errors  []string
}

type WorkflowPushStats struct {
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
	defaultDir := filepath.Join(".", "suprsend", "workflows")
	fmt.Fprintf(os.Stdout, "Where would you like to save the workflows?\n")
	fmt.Fprintf(os.Stdout, "Default: %s\n", defaultDir)
	fmt.Fprintf(os.Stdout, "Enter directory path (or press Enter for default): ")
	input, _ := reader.ReadString('\n')
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

func validateInputDirectory(dirPath string) error {
	info, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("directory does not exist: %s", dirPath)
		}
		return fmt.Errorf("error checking directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path '%s' exists but is not a directory", dirPath)
	}
	if info.Mode().Perm()&0o400 == 0 {
		return fmt.Errorf("directory '%s' is not readable", dirPath)
	}
	return nil
}


func WriteWorkflowsToFiles(resp mgmnt.WorkflowsResponse, outputDir string) (*WorkflowWriteStats, error) {
	stats := &WorkflowWriteStats{
		Total:  len(resp.Results),
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

	for _, wf := range resp.Results {
		obj, ok := wf.(map[string]any)
		if !ok {
			stats.Failed++
			stats.Errors = append(stats.Errors, "Invalid workflow format")
			continue
		}

		slug, _ := obj["slug"].(string)
		slugDir := filepath.Join(outputDir, slug)
		if err := os.MkdirAll(slugDir, 0o755); err != nil {
			stats.Failed++
			stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to create directory for '%s': %v", slug, err))
			continue
		}

		fileData, err := json.MarshalIndent(obj, "", "  ")
		if err != nil {
			log.Errorf("Failed to marshal workflow '%s': %v", slug, err)
			stats.Failed++
			stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to marshal workflow '%s': %v", slug, err))
			continue
		}

		filename := filepath.Join(slugDir, "workflow.json")
		if err := os.WriteFile(filename, append(fileData, '\n'), 0o644); err != nil {
			log.Errorf("Failed to write file '%s': %v", filename, err)
			stats.Failed++
			stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to write file '%s': %v", filename, err))
			continue
		}

		log.Infof("Wrote workflow to %s", filename)
		stats.Success++
	}

	return stats, nil
}
