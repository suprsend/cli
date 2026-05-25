package translation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/utils"
)

var translationPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull preference translations",
	Long:  "Download preference category translations from a workspace to local JSON files. Creates one file per locale (e.g., es.json, fr.json) in the output directory.",
	Example: `  # Pull all locale translations to default directory
  suprsend category translation pull

  # Pull to a custom directory
  suprsend category translation pull --dir ./my-categories

  # Pull in the production workspace
  suprsend category translation pull --workspace production`,
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		outputDir, _ := cmd.Flags().GetString("dir")
		force, _ := cmd.Flags().GetBool("force")

		return PullTranslations(cmd.Context(), workspace, outputDir, force)
	},
}

func PullTranslations(ctx context.Context, workspace, outputDir string, force bool) error {
	if workspace == "" {
		return clierr.New("workspace flag is required", clierr.CodeInvalidUsage)
	}

	if outputDir == "" {
		outputDir = defaultDir
		if _, err := os.Stat(outputDir); os.IsNotExist(err) {
			if force {
				log.Infof("Using default directory: %s", outputDir)
			} else {
				od, success := promptForOutputDirectory()
				if !success {
					return nil
				}
				outputDir = od
			}
		}
		if outputDir == "" {
			log.Info("No output directory specified. Exiting.")
			return nil
		}
	}
	if err := ensureOutputDirectory(outputDir); err != nil {
		return fmt.Errorf("error with output directory: %w", err)
	}

	spinner := utils.NewSpinner("Loading...")

	mgmntClient := utils.GetSuprSendMgmntClient()
	locales, err := mgmntClient.ListPreferenceTranslations(ctx, workspace)
	if err != nil {
		return fmt.Errorf("couldn't fetch translation locales from workspace '%s': %w", workspace, err)
	}

	successCount := 0
	failedCount := 0
	var errors []string

	for _, localeResult := range locales.Results {
		locale := localeResult.Locale
		translations, err := mgmntClient.GetPreferenceTranslationsForLocale(ctx, workspace, locale)
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
		// Per-file write line, matching schema/workflow pull's "Wrote ... to ..."
		// shape so users can see what landed on disk and can grep / diff
		// against expectations.
		log.Infof("Wrote translation to %s", filename)
	}

	spinner.Stop(fmt.Sprintf("Pulled %d translation(s) from %s", successCount, workspace))

	log.Info("=== Translation Pull Summary ===")
	log.Infof("Total locales processed: %d", len(locales.Results))
	log.Infof("Successfully written: %d", successCount)
	if failedCount > 0 {
		log.Infof("Failed to write: %d", failedCount)
		log.Info("Failed translations:")
		for _, errMsg := range errors {
			log.Infof("  - %s", errMsg)
		}
	}
	return nil
}

func init() {
	translationPullCmd.Flags().StringP("dir", "d", "", "Directory to save translation files to (default: "+defaultDir+")")
	translationPullCmd.Flags().BoolP("force", "F", false, "Skip directory confirmation prompt, use default path")
	TranslationCmd.AddCommand(translationPullCmd)
}
