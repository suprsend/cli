package translation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/cli/mgmnt"
)

// PushTranslationStats summarises a translation push so callers (notably
// category push) can fold the numbers into their own end-of-run summary.
type PushTranslationStats struct {
	Total          int      // locales attempted (excluding English)
	Success        int      // locales successfully pushed
	Failed         int      // locales that errored
	SkippedEnglish int      // english files skipped (informational)
	Errors         []string // human-readable failure descriptions
}

var translationPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push preference translations",
	Long:  "Upload local preference category translation files to a workspace. Reads {locale}.json files directly from the input directory (default `./suprsend/preference_categories/translations`). English (en.json) is skipped — the API rejects pushes to the source-of-truth locale. Translations have no draft state: pushed files take effect immediately. Use --locale to push a single locale, or omit to push all.",
	Example: `  # Push all locale translations from default directory
  suprsend category translation push

  # Push a specific locale only
  suprsend category translation push --locale es

  # Push from a custom directory (the directory that contains the locale files)
  suprsend category translation push --dir ./my-categories/translations

  # Dry run: preview what would be pushed without making changes
  suprsend category translation push --dry-run`,
	Annotations: map[string]string{
		"skills:tip.a-result": "Each locale is reported on stdout as `pushed` (translations go live immediately), `skipped` (English) or `failed`, followed by a summary; with `-o json` the result is one JSON object with `items` and `counts`.",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		locale, _ := cmd.Flags().GetString("locale")
		dir, _ := cmd.Flags().GetString("dir")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		report := utils.NewReport("category translation push", workspace, dryRun)
		_, err := PushTranslations(report, workspace, locale, dir, dryRun)
		return report.Finish(err)
	},
}

// englishLocaleFilename is the on-disk filename for the source-of-truth
// English translation that the management API refuses to accept on push.
const englishLocaleFilename = "en.json"

// PushTranslations uploads non-English locale translation files from dir to
// the workspace and adds one report item per locale file. Returns stats so
// the caller can tell what happened.
func PushTranslations(report *utils.Report, workspace, locale, dir string, dryRun bool) (*PushTranslationStats, error) {
	if workspace == "" {
		return nil, clierr.New("workspace flag is required", clierr.CodeInvalidUsage)
	}
	if locale == "en" {
		return nil, clierr.New("cannot push English translations — the API treats en.json as the source of truth", clierr.CodeInvalidUsage)
	}

	// Determine the translations directory
	translationsDir := defaultDir
	if dir != "" {
		translationsDir = dir
	}

	if _, err := os.Stat(translationsDir); os.IsNotExist(err) {
		return nil, clierr.New(fmt.Sprintf("directory %s does not exist", translationsDir), clierr.CodeFileNotFound)
	}

	files, err := os.ReadDir(translationsDir)
	if err != nil {
		return nil, clierr.Wrap(err, clierr.CodeFileNotFound, fmt.Sprintf("couldn't read directory %s", translationsDir))
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

	stats := &PushTranslationStats{
		Total:          len(pushable),
		SkippedEnglish: len(skippedEnglish),
	}

	if len(pushable) == 0 {
		// Distinguish "directory has nothing JSON-like" from "directory has
		// only English". The latter is the common foot-gun: the user pulled
		// from a workspace whose only locale is the source of truth, then
		// expected push to do something.
		if len(skippedEnglish) > 0 {
			return stats, clierr.New(
				fmt.Sprintf("nothing to push from %s — only English translations found, which the API rejects. Pull from a workspace with non-English locales, or add e.g. es.json / fr.json to the directory.", translationsDir),
				clierr.CodeInvalidUsage,
			)
		}
		if locale != "" {
			return stats, clierr.New(fmt.Sprintf("no %s.json found in %s", locale, translationsDir), clierr.CodeFileNotFound)
		}
		return stats, clierr.New(fmt.Sprintf("no locale JSON files found in %s", translationsDir), clierr.CodeFileNotFound)
	}

	for _, fileName := range skippedEnglish {
		report.Add(translationResource(fileName), utils.StatusSkipped, "English is the source of truth and can't be pushed")
	}

	if dryRun {
		for _, fileName := range pushable {
			report.Add(translationResource(fileName), utils.StatusWouldPush, "")
		}
		return stats, nil
	}

	mgmntClient := utils.GetSuprSendMgmntClient()
	fail := func(fileName, action string, err error) {
		stats.Failed++
		stats.Errors = append(stats.Errors, fmt.Sprintf("%s: %s: %v", translationResource(fileName), action, err))
		report.Fail(translationResource(fileName), err)
	}

	for _, fileName := range pushable {
		filePath := filepath.Join(translationsDir, fileName)
		fileLocale := strings.TrimSuffix(fileName, ".json")

		data, err := os.ReadFile(filePath)
		if err != nil {
			fail(fileName, "failed to read", err)
			continue
		}

		var translation mgmnt.PreferenceTranslationContent
		if err := json.Unmarshal(data, &translation); err != nil {
			fail(fileName, "failed to parse", err)
			continue
		}

		spinner := utils.NewSpinner(fmt.Sprintf("Pushing %s...", fileName))
		err = mgmntClient.PushPreferenceTranslation(workspace, fileLocale, translation)
		spinner.Stop("")
		if err != nil {
			fail(fileName, "failed to push", err)
			continue
		}

		stats.Success++
		report.Add(translationResource(fileName), utils.StatusPushed, "live")
	}

	if stats.Failed > 0 {
		return stats, clierr.Failures(fmt.Sprintf("%d locale(s) failed to push", stats.Failed), stats.Errors, clierr.CodeAPIInternal)
	}
	return stats, nil
}

func translationResource(fileName string) string {
	return "preference_categories/translations/" + fileName
}

func init() {
	translationPushCmd.Flags().String("locale", "", "Locale code to push, e.g., es, fr (omit to push all)")
	translationPushCmd.Flags().StringP("dir", "d", "", "Directory containing translation JSON files (default: "+defaultDir+")")
	translationPushCmd.Flags().BoolP("dry-run", "n", false, "Report what would be pushed without making any changes")
	TranslationCmd.AddCommand(translationPushCmd)
}
