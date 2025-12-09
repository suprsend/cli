package translation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/utils"
	"github.com/yarlson/pin"
)

var translationPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull preference translations",
	Long:  "Pull preference translations",
	Run: func(cmd *cobra.Command, args []string) {
		workspace, _ := cmd.Flags().GetString("workspace")
		outputDir, _ := cmd.Flags().GetString("dir")
		force, _ := cmd.Flags().GetBool("force")

		PullTranslations(workspace, outputDir, force)
	},
}

func PullTranslations(workspace, outputDir string, force bool) {
	if workspace == "" {
		log.Error("workspace flag is required")
		return
	}

	if outputDir == "" {
		outputDir = defaultDir
		if _, err := os.Stat(outputDir); os.IsNotExist(err) {
			if force {
				fmt.Fprintf(os.Stdout, "Using default directory: %s\n", outputDir)
			} else {
				outputDir = promptForOutputDirectory()
			}
		}
		if outputDir == "" {
			fmt.Fprintf(os.Stdout, "No output directory specified. Exiting.\n")
			return
		}
	}
	if err := ensureOutputDirectory(outputDir); err != nil {
		log.WithError(err).Errorf("Error with output directory: %v", err)
		return
	}

	var p *pin.Pin
	if !utils.IsOutputPiped() {
		p = pin.New("Loading...",
			pin.WithSpinnerColor(pin.ColorCyan),
			pin.WithTextColor(pin.ColorYellow),
		)
		cancel := p.Start(context.Background())
		defer cancel()
	}

	mgmntClient := utils.GetSuprSendMgmntClient()
	locales, err := mgmntClient.ListPreferenceTranslations(workspace)
	if err != nil {
		log.WithError(err).Errorf("Couldn't fetch translation locales from workspace '%s'", workspace)
		return
	}

	successCount := 0
	failedCount := 0
	var errors []string

	for _, localeResult := range locales.Results {
		locale := localeResult.Locale
		translations, err := mgmntClient.GetPreferenceTranslationsForLocale(workspace, locale)
		if err != nil {
			log.WithError(err).Errorf("Couldn't fetch translations for locale '%s' from workspace '%s'", locale, workspace)
			failedCount++
			errors = append(errors, fmt.Sprintf("Locale '%s': Failed to fetch translations from workspace '%s' - %v", locale, workspace, err))
			continue
		}

		filename := filepath.Join(outputDir, fmt.Sprintf("%s.json", locale))
		fileData, err := json.MarshalIndent(translations, "", "  ")
		if err != nil {
			log.WithError(err).Errorf("Failed to serialize translations to JSON for locale '%s'", locale)
			failedCount++
			errors = append(errors, fmt.Sprintf("Failed to serialize translations to JSON for locale '%s': %v", locale, err))
			continue
		}

		if err := os.WriteFile(filename, fileData, 0644); err != nil {
			log.WithError(err).Errorf("Couldn't write file '%s' for locale '%s'", filename, locale)
			failedCount++
			errors = append(errors, fmt.Sprintf("Locale '%s': Failed to write file '%s' - %v", locale, filename, err))
			continue
		}
		successCount++
	}

	if p != nil {
		p.Stop(fmt.Sprintf("Pulled translations from %s", workspace))
	}

	fmt.Fprintf(os.Stdout, "\n=== Translation Pull Summary ===\n")
	fmt.Fprintf(os.Stdout, "Total locales processed: %d\n", len(locales.Results))
	fmt.Fprintf(os.Stdout, "Successfully written: %d\n", successCount)
	if failedCount > 0 {
		fmt.Fprintf(os.Stdout, "Failed to write: %d\n", failedCount)
		fmt.Fprintf(os.Stdout, "\nFailed translations:\n")
		for _, errMsg := range errors {
			fmt.Fprintf(os.Stdout, "  - %s\n", errMsg)
		}
	}
}

func init() {
	translationPullCmd.Flags().StringP("dir", "d", "", "Output directory for translations (default: "+defaultDir+")")
	translationPullCmd.Flags().BoolP("force", "f", false, "Force using default directory without prompting")
	TranslationCmd.AddCommand(translationPullCmd)
}
