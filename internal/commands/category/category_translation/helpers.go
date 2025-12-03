package category_translation

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var defaultDir = filepath.Join(".", "suprsend", "category")

func promptForOutputDirectory() string {
	reader := bufio.NewReader(os.Stdin)
	defaultDir := defaultDir
	fmt.Fprintf(os.Stdout, "Where would you like to save the translations?\n")
	fmt.Fprintf(os.Stdout, "Default: %s\n", defaultDir)
	fmt.Fprintf(os.Stdout, "Enter directory path (or press Enter for default): ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultDir
	}
	return input
}

func ensureOutputDirectory(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	return nil
}
