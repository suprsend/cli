package translation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/cli/mgmnt"
)

var translationPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push preference translations",
	Long:  "Upload local preference category translation files to a workspace. Reads {locale}.json files from the input directory. English (en.json) is skipped — the API rejects pushes to the source-of-truth locale. Use --locale to push a single locale, or omit to push all.",
	Example: `  # Push all locale translations from default directory
  suprsend category translation push

  # Push a specific locale only
  suprsend category translation push --locale es

  # Push from a custom directory
  suprsend category translation push --dir ./my-categories

  # Dry run: preview what would be pushed without making changes
  suprsend category translation push --dry-run`,
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		locale, _ := cmd.Flags().GetString("locale")
		dir, _ := cmd.Flags().GetString("dir")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		return PushTranslations(workspace, locale, dir, dryRun)
	},
}

// englishLocaleFilename is the on-disk filename for the source-of-truth
// English translation that the management API refuses to accept on push.
const englishLocaleFilename = "en.json"

func PushTranslations(workspace, locale, dir string, dryRun bool) error {
	if workspace == "" {
		return clierr.New("workspace flag is required", clierr.CodeInvalidUsage)
	}
	if locale == "en" {
		return clierr.New("cannot push English translations — the API treats en.json as the source of truth", clierr.CodeInvalidUsage)
	}

	// Determine the translations directory
	translationsDir := defaultDir
	if dir != "" {
		translationsDir = dir
	}

	if _, err := os.Stat(translationsDir); os.IsNotExist(err) {
		return clierr.New(fmt.Sprintf("directory %s does not exist", translationsDir), clierr.CodeFileNotFound)
	}

	files, err := os.ReadDir(translationsDir)
	if err != nil {
		return clierr.Wrap(err, clierr.CodeFileNotFound, fmt.Sprintf("couldn't read directory %s", translationsDir))
	}

	// Filter for locale JSON files (e.g., en.json, es.json) and split into
	// pushable and skipped buckets so we can be explicit about both.
	var pushable []string
	var skippedEnglish []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		fileLocale := strings.TrimSuffix(name, ".json")
		if fileLocale == "categories" {
			continue
		}
		if locale != "" && fileLocale != locale {
			continue
		}
		if name == englishLocaleFilename {
			skippedEnglish = append(skippedEnglish, name)
			continue
		}
		pushable = append(pushable, name)
	}

	if len(pushable) == 0 {
		// Distinguish "directory has nothing JSON-like" from "directory has
		// only English". The latter is the common foot-gun: the user pulled
		// from a workspace whose only locale is the source of truth, then
		// expected push to do something.
		if len(skippedEnglish) > 0 {
			return clierr.New(
				fmt.Sprintf("nothing to push from %s — only English translations found, which the API rejects. Pull from a workspace with non-English locales, or add e.g. es.json / fr.json to the directory.", translationsDir),
				clierr.CodeInvalidUsage,
			)
		}
		if locale != "" {
			return clierr.New(fmt.Sprintf("no %s.json found in %s", locale, translationsDir), clierr.CodeFileNotFound)
		}
		return clierr.New(fmt.Sprintf("no locale JSON files found in %s", translationsDir), clierr.CodeFileNotFound)
	}

	if dryRun {
		log.Infof("DRY RUN: would push %d translation file(s) to %s", len(pushable), workspace)
		for _, fileName := range pushable {
			log.Infof("  - %s", strings.TrimSuffix(fileName, ".json"))
		}
		for _, fileName := range skippedEnglish {
			log.Infof("  - %s (skipped: English source of truth)", strings.TrimSuffix(fileName, ".json"))
		}
		return nil
	}

	for _, fileName := range skippedEnglish {
		log.Infof("Skipping %s — English translations are the source of truth and cannot be pushed.", fileName)
	}

	spinner := utils.NewSpinner("Pushing translations...")
	mgmntClient := utils.GetSuprSendMgmntClient()

	successCount := 0
	failedCount := 0
	var errors []string

	for _, fileName := range pushable {
		filePath := filepath.Join(translationsDir, fileName)
		fileLocale := strings.TrimSuffix(fileName, ".json")

		data, err := os.ReadFile(filePath)
		if err != nil {
			log.WithError(err).Debugf("Couldn't read translations from file %s", filePath)
			failedCount++
			errors = append(errors, fmt.Sprintf("Failed to read %s: %v", fileName, err))
			continue
		}

		var translation mgmnt.PreferenceTranslationContent
		if err := json.Unmarshal(data, &translation); err != nil {
			log.WithError(err).Errorf("Couldn't parse translations JSON from %s", filePath)
			failedCount++
			errors = append(errors, fmt.Sprintf("Failed to parse JSON from %s: %v", fileName, err))
			continue
		}

		if err := mgmntClient.PushPreferenceTranslation(workspace, fileLocale, translation); err != nil {
			log.WithError(err).Errorf("Couldn't push translations for locale %s", fileLocale)
			failedCount++
			errors = append(errors, fmt.Sprintf("Failed to push %s: %v", fileName, err))
			continue
		}

		successCount++
		log.Infof("Successfully pushed translations for locale: %s", fileLocale)
	}

	msg := fmt.Sprintf("Pushed %d translation file(s) to %s", successCount, workspace)
	if failedCount > 0 {
		msg += fmt.Sprintf(" (%d failed)", failedCount)
	}
	spinner.Stop(msg)

	if len(errors) > 0 {
		log.Info("Errors:")
		for _, errMsg := range errors {
			log.Infof("  - %s", errMsg)
		}
		return clierr.New(fmt.Sprintf("%d locale(s) failed to push", failedCount), clierr.CodeAPIInternal)
	}
	return nil
}

func init() {
	translationPushCmd.Flags().String("locale", "", "Locale code to push, e.g., es, fr (omit to push all)")
	translationPushCmd.Flags().StringP("dir", "d", "", "Directory containing translation JSON files (default: "+defaultDir+")")
	translationPushCmd.Flags().BoolP("dry-run", "n", false, "Print what would be pushed without making any changes")
	TranslationCmd.AddCommand(translationPushCmd)
}
