package category

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

const defaultCategoryDir = "suprsend/preference_categories"

// categoriesOnDisk is the on-disk format for categories.json.
// Only editable fields are stored; server-side readonly fields (hash, version_no, status, etc.) are excluded.
// $schema is preserved from the API response.
type categoriesOnDisk struct {
	Schema         string               `json:"$schema,omitempty"`
	RootCategories []mgmnt.RootCategory `json:"root_categories"`
}

func writeCategoriesFile(resp *mgmnt.PreferenceCategoryResponse, filePath string) error {
	data := categoriesOnDisk{
		Schema:         resp.Schema,
		RootCategories: resp.RootCategories,
	}
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal categories: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return fmt.Errorf("failed to ensure directory %s: %w", filepath.Dir(filePath), err)
	}
	log.Infof("Successfully wrote categories to %s", filePath)
	return os.WriteFile(filePath, jsonData, 0644)
}

func promptForOutputDirectory() (string, bool) {
	if !utils.IsInputInteractive() {
		fmt.Fprintf(os.Stderr, "required flag missing, cannot prompt in non-interactive mode")
		return "", false
	}
	reader := bufio.NewReader(os.Stdin)
	defaultDir := filepath.Join(".", defaultCategoryDir)
	fmt.Fprintf(os.Stdout, "Where would you like to save the categories?\n")
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

func WriteToFileWithPath(data interface{}, filePath string) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return fmt.Errorf("failed to ensure directory %s: %w", filepath.Dir(filePath), err)
	}
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}
	log.Infof("Successfully wrote categories to %s", filePath)
	return os.WriteFile(filePath, jsonData, 0644)
}

func WriteToFile(data interface{}, filePath string) error {
	return WriteToFileWithPath(data, filePath)
}

func ReadFromFile(filepath string) (interface{}, error) {
	jsonData, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	var data interface{}
	err = json.Unmarshal(jsonData, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %w", err)
	}
	return data, nil
}

func ensureOutputDirectory(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	return nil
}
