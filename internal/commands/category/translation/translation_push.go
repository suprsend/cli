package translation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/cli/mgmnt"
	"github.com/yarlson/pin"
)

var translationPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push preference translations",
	Long:  "Push preference translations",
	Run: func(cmd *cobra.Command, args []string) {
		workspace, _ := cmd.Flags().GetString("workspace")
		locale, _ := cmd.Flags().GetString("locale")
		dir, _ := cmd.Flags().GetString("dir")

		PushTranslations(workspace, locale, dir)
	},
}

func PushTranslations(workspace, locale, dir string) {
	if workspace == "" {
		log.Error("workspace flag is required")
		return
	}

	// Determine the translations directory
	translationsDir := defaultDir
	if dir != "" {
		// If a directory is provided, use it
		translationsDir = dir
	}

	// Check if directory exists
	if _, err := os.Stat(translationsDir); os.IsNotExist(err) {
		log.Errorf("Directory %s does not exist", translationsDir)
		return
	}

	// Read all files in the translations directory
	files, err := os.ReadDir(translationsDir)
	if err != nil {
		log.WithError(err).Errorf("Couldn't read directory %s", translationsDir)
		return
	}

	if locale == "en" {
		log.Warnf("cannot push English translations")
		return
	}

	// Filter for locale JSON files (e.g., en.json, es.json)
	var localeFiles []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		// Check if file matches pattern {locale}.json
		if strings.HasSuffix(name, ".json") {
			// Extract locale from filename (remove .json extension)
			fileLocale := strings.TrimSuffix(name, ".json")
			// If a specific locale is requested, filter for it
			if locale != "" && fileLocale != locale {
				continue
			}
			// Skip categories_preferences.json file
			if fileLocale == "categories_preferences" {
				continue
			}
			localeFiles = append(localeFiles, name)
		}
	}

	if len(localeFiles) == 0 {
		log.Warnf("No locale JSON files found in %s", translationsDir)
		return
	}

	var p *pin.Pin
	if !utils.IsOutputPiped() {
		p = pin.New("Pushing translations...",
			pin.WithSpinnerColor(pin.ColorCyan),
			pin.WithTextColor(pin.ColorYellow),
		)
		cancel := p.Start(context.Background())
		defer cancel()
	}

	mgmntClient := utils.GetSuprSendMgmntClient()

	// Track success/failure stats
	successCount := 0
	failedCount := 0
	var errors []string

	// Iterate through each locale file and push translations
	for _, fileName := range localeFiles {
		// Skip English translations
		if fileName == "en.json" {
			continue
		}
		filePath := filepath.Join(translationsDir, fileName)
		fileLocale := strings.TrimSuffix(fileName, ".json")

		// Read the translation file
		data, err := os.ReadFile(filePath)
		if err != nil {
			log.WithError(err).Debugf("Couldn't read translations from file %s", filePath)
			failedCount++
			errors = append(errors, fmt.Sprintf("Failed to read %s: %v", fileName, err))
			continue
		}

		// Parse the JSON
		var translation mgmnt.PreferenceTranslationContent
		if err := json.Unmarshal(data, &translation); err != nil {
			log.WithError(err).Errorf("Couldn't parse translations JSON from %s", filePath)
			failedCount++
			errors = append(errors, fmt.Sprintf("Failed to parse JSON from %s: %v", fileName, err))
			continue
		}

		// Push translations for this locale
		err = mgmntClient.PushPreferenceTranslation(workspace, fileLocale, translation)
		if err != nil {
			log.WithError(err).Errorf("Couldn't push translations for locale %s", fileLocale)
			failedCount++
			errors = append(errors, fmt.Sprintf("Failed to push %s: %v", fileName, err))
			continue
		}

		successCount++
		log.Infof("Successfully pushed translations for locale: %s", fileLocale)
	}

	if p != nil {
		msg := fmt.Sprintf("Pushed %d translation file(s) to %s", successCount, workspace)
		if failedCount > 0 {
			msg += fmt.Sprintf(" (%d failed)", failedCount)
		}
		p.Stop(msg)
	}

	// Print errors if any
	if len(errors) > 0 {
		fmt.Fprintf(os.Stdout, "\nErrors:\n")
		for _, errMsg := range errors {
			fmt.Fprintf(os.Stdout, "  - %s\n", errMsg)
		}
	}
}

func init() {
	translationPushCmd.Flags().StringP("locale", "l", "", "Specific locale to push (if not provided, all locale files will be pushed)")
	translationPushCmd.Flags().StringP("dir", "d", "", "Directory for translations to push from (default: "+defaultDir+")")
	TranslationCmd.AddCommand(translationPushCmd)
}
