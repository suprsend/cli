package translation

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/suprsend/cli/internal/utils"
)

var defaultDir = filepath.Join(".", "suprsend", "preference_categories", "translations")

func promptForOutputDirectory() (string, bool) {
	if !utils.IsInputInteractive() {
		fmt.Fprintf(os.Stderr, "required flag missing, cannot prompt in non-interactive mode")
		return "", false
	}
	reader := bufio.NewReader(os.Stdin)
	dd := defaultDir
	fmt.Fprintf(os.Stdout, "Where would you like to save the translations?\n")
	fmt.Fprintf(os.Stdout, "Default: %s\n", dd)
	fmt.Fprintf(os.Stdout, "Enter directory path (or press Enter for default): ")
	input, err := reader.ReadString('\n')
	if err != nil {
		// If there's an error reading input, fall back to default directory
		fmt.Fprintf(os.Stderr, "Error reading input: %v. Using default directory: %s\n", err, dd)
		return dd, true
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return dd, true
	}
	return input, true
}

func ensureOutputDirectory(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	return nil
}
