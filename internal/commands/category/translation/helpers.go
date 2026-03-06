package translation

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var defaultDir = filepath.Join(".", "suprsend", "category", "translation")

func promptForOutputDirectory() string {
	reader := bufio.NewReader(os.Stdin)
	dd := defaultDir
	fmt.Fprintf(os.Stdout, "Where would you like to save the translations?\n")
	fmt.Fprintf(os.Stdout, "Default: %s\n", dd)
	fmt.Fprintf(os.Stdout, "Enter directory path (or press Enter for default): ")
	input, err := reader.ReadString('\n')
	if err != nil {
		// If there's an error reading input, fall back to default directory
		fmt.Fprintf(os.Stderr, "Error reading input: %v. Using default directory: %s\n", err, dd)
		return dd
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return dd
	}
	return input
}

func ensureOutputDirectory(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	return nil
}
