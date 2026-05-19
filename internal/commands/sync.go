package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/commands/category"
	"github.com/suprsend/cli/internal/commands/event"
	"github.com/suprsend/cli/internal/commands/schema"
	"github.com/suprsend/cli/internal/commands/template"
	"github.com/suprsend/cli/internal/commands/translation"
	"github.com/suprsend/cli/internal/commands/workflow"
	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/cli/mgmnt"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync SuprSend assets from one workspace to another",
	Long:  `Sync notification assets from one workspace to another. Pulls assets from the source workspace and pushes them to the destination. Supports syncing all asset types or a specific type (workflow, schema, event, category, translation, template). Source and destination workspaces must be different.`,
	Example: `  # Sync all assets from staging to production
  suprsend sync --from staging --to production

  # Sync only workflows
  suprsend sync --from staging --to production --assets workflow

  # Sync and commit immediately (prompts for confirmation)
  suprsend sync --from staging --to production --commit

  # Dry run: preview what would be synced without making changes
  suprsend sync --from staging --to production --dry-run`,
	Annotations: map[string]string{
		"skills:tip.a-direction": "`--from` is the source, `--to` is the destination. They must be different workspaces; sync **overwrites** drafts in the destination.",
		"skills:tip.b-dryrun":    "Pair with `--dry-run` to validate every asset server-side without writing to the destination. Add `--assets <type>` to scope to one resource type (workflow / schema / event / category / translation / template).",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		mode, _ := cmd.Flags().GetString("mode")
		fromWorkspace, _ := cmd.Flags().GetString("from")
		toWorkspace, _ := cmd.Flags().GetString("to")
		assets, _ := cmd.Flags().GetString("assets")
		dirPath, _ := cmd.Flags().GetString("dir")
		commit, _ := cmd.Flags().GetBool("commit")
		commitMessage, _ := cmd.Flags().GetString("commit-message")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		force, _ := cmd.Flags().GetBool("force")

		if fromWorkspace == toWorkspace {
			return clierr.New("cannot sync within the same workspace; source and destination workspaces must be different", clierr.CodeInvalidUsage)
		}

		var assetsToSync []string
		switch assets {
		case "all":
			assetsToSync = []string{"category", "schema", "event", "template", "workflow", "translation"}
		case "workflow":
			assetsToSync = []string{"workflow"}
		case "schema":
			assetsToSync = []string{"schema"}
		case "event":
			assetsToSync = []string{"event"}
		case "category":
			assetsToSync = []string{"category"}
		case "translation":
			assetsToSync = []string{"translation"}
		case "template":
			assetsToSync = []string{"template"}
		default:
			return clierr.New(
				fmt.Sprintf("invalid asset type: '%s'; valid options are: all, workflow, schema, event, category, translation, template", assets),
				clierr.CodeInvalidUsage,
			)
		}

		if commit && !dryRun && !force {
			msg := fmt.Sprintf("This will sync %s from \"%s\" to \"%s\" and commit each. Continue?", assets, fromWorkspace, toWorkspace)
			confirmed, err := utils.ConfirmDestructiveAction(msg)
			if err != nil || !confirmed {
				log.Info("Aborted.")
				return nil
			}
		}

		log.Infof("Syncing assets from %s to %s ...", fromWorkspace, toWorkspace)
		log.Infof("Assets to sync: %v", assetsToSync)

		mgmntClient := utils.GetSuprSendMgmntClient()
		hasErrors := false

		for _, assetType := range assetsToSync {
			switch assetType {
			case "workflow":
				err := syncWorkflows(mgmntClient, fromWorkspace, toWorkspace, mode, dirPath, commit, commitMessage, dryRun)
				if err != nil {
					log.Errorf("Failed to sync workflows: %v", err)
					hasErrors = true
				}
			case "schema":
				err := syncSchemas(mgmntClient, fromWorkspace, toWorkspace, mode, dirPath, commit, commitMessage, dryRun)
				if err != nil {
					log.Errorf("Failed to sync schemas: %v", err)
					hasErrors = true
				}
			case "event":
				err := syncEvents(mgmntClient, fromWorkspace, toWorkspace, dirPath, dryRun)
				if err != nil {
					log.Errorf("Failed to sync events: %v", err)
					hasErrors = true
				}
			case "category":
				err := syncCategories(mgmntClient, fromWorkspace, toWorkspace, mode, dirPath, commit, commitMessage, dryRun)
				if err != nil {
					log.Errorf("Failed to sync categories: %v", err)
					hasErrors = true
				}
			case "translation":
				err := syncTranslation(mgmntClient, fromWorkspace, toWorkspace, mode, dirPath, commit, commitMessage, dryRun)
				if err != nil {
					log.Errorf("Failed to sync translations: %v", err)
					hasErrors = true
				}
			case "template":
				err := syncTemplates(mgmntClient, fromWorkspace, toWorkspace, mode, dirPath, commit, commitMessage, dryRun)
				if err != nil {
					log.Errorf("Failed to sync templates: %v", err)
					hasErrors = true
				}
			}
		}
		if hasErrors {
			return clierr.New("sync complete with errors", clierr.CodeAPIInternal)
		}
		log.Info("Sync complete")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)

	// Flags consumed in Run
	syncCmd.Flags().StringP("from", "S", "staging", "Source workspace to pull assets from")
	syncCmd.Flags().StringP("to", "t", "production", "Destination workspace to push assets to")
	syncCmd.Flags().StringP("dir", "d", "", "Local directory for intermediate file storage during sync")
	syncCmd.Flags().StringP("mode", "m", "live", "Version mode: draft or live")
	syncCmd.Flags().StringP("assets", "a", "all", "Asset types to sync: all, workflow, schema, event, category, translation, or template")
	syncCmd.Flags().BoolP("commit", "c", true, "Promote changes from draft to live after syncing")
	syncCmd.Flags().String("commit-message", "", "Commit message applied to every committed resource in this sync run (required when --commit is set)")
	syncCmd.Flags().BoolP("dry-run", "n", false, "Print what would be synced without making any changes")
	syncCmd.Flags().BoolP("force", "F", false, "Skip confirmation prompt")
}

func syncWorkflows(mgmntClient *mgmnt.SS_MgmntClient, fromWorkspace, toWorkspace, mode, dirPath string, commit bool, commitMessage string, dryRun bool) error {
	if dirPath == "" {
		dirPath = filepath.Join(".", "suprsend", "workflows")
	} else {
		dirPath = filepath.Join(dirPath, "workflows")
	}

	spinner := utils.NewSpinner(fmt.Sprintf("Pulling workflows from %s ...", fromWorkspace))

	workflows_resp, err := mgmntClient.GetWorkflows(fromWorkspace, mode)
	if err != nil {
		spinner.Stop("")
		return clierr.Wrap(err, clierr.CodeAPIInternal, "error getting workflows")
	}

	_, err = workflow.WriteWorkflowsToFiles(*workflows_resp, dirPath)
	if err != nil {
		spinner.Stop("")
		return clierr.Wrap(err, clierr.CodeFileParseFailed, "error writing workflows to files")
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		spinner.Stop("")
		return clierr.Wrap(err, clierr.CodeFileNotFound, "error reading local workflows directory")
	}

	spinner.UpdateMessage(fmt.Sprintf("Pushing workflows to %s ...", toWorkspace))

	var errors []string
	successCount := 0
	dryRunCount := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		slug := entry.Name()
		path := filepath.Join(dirPath, slug, "workflow.json")
		data, err := os.ReadFile(path)
		if err != nil {
			errors = append(errors, fmt.Sprintf("error reading file %s: %v", path, err))
			continue
		}

		var wf map[string]any
		if err := json.Unmarshal(data, &wf); err != nil {
			errors = append(errors, fmt.Sprintf("error unmarshalling JSON for %s: %v", path, err))
			continue
		}

		wf["slug"] = slug

		if dryRun {
			dryRunCount++
			log.Infof("DRY RUN: would push workflow %s to %s", slug, toWorkspace)
			continue
		}

		err = mgmntClient.PushWorkflow(toWorkspace, slug, wf, commit, commitMessage)
		if err != nil {
			errors = append(errors, fmt.Sprintf("workflows/%s: failed to push: %v", slug, err))
			log.Errorf("workflows/%s: failed to push: %v", slug, err)
			continue
		}

		successCount++
		log.Infof("Pushed workflow: %s", slug)
	}
	if len(errors) > 0 {
		spinner.Stop(fmt.Sprintf("Synced workflows with %d error(s)", len(errors)))
		return clierr.New(fmt.Sprintf("one or more workflows failed to sync:\n%s", strings.Join(errors, "\n")), clierr.CodeAPIInternal)
	}
	if dryRun {
		spinner.Stop(fmt.Sprintf("DRY RUN: would push %d workflow(s) to %s", dryRunCount, toWorkspace))
	} else {
		spinner.Stop(fmt.Sprintf("Synced %d workflow(s) to %s", successCount, toWorkspace))
	}
	return nil
}

func syncSchemas(mgmntClient *mgmnt.SS_MgmntClient, fromWorkspace, toWorkspace, mode, dirPath string, commit bool, commitMessage string, dryRun bool) error {
	if dirPath == "" {
		dirPath = filepath.Join(".", "suprsend", "schemas")
	} else {
		dirPath = filepath.Join(dirPath, "schemas")
	}

	spinner := utils.NewSpinner(fmt.Sprintf("Pulling schemas from %s ...", fromWorkspace))

	schemas_resp, err := mgmntClient.GetSchemas(fromWorkspace, mode)
	if err != nil {
		spinner.Stop("")
		return clierr.Wrap(err, clierr.CodeAPIInternal, "error getting schemas")
	}

	_, err = schema.WriteSchemasToFiles(schemas_resp, dirPath)
	if err != nil {
		spinner.Stop("")
		return clierr.Wrap(err, clierr.CodeFileParseFailed, "error writing schemas to files")
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		spinner.Stop("")
		return clierr.Wrap(err, clierr.CodeFileNotFound, "error reading local schemas directory")
	}

	spinner.UpdateMessage(fmt.Sprintf("Pushing schemas to %s ...", toWorkspace))

	var errors []string
	successCount := 0
	dryRunCount := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		slug := entry.Name()
		sch, err := schema.ReadAndMergeSchemaFiles(filepath.Join(dirPath, slug), slug)
		if err != nil {
			errors = append(errors, err.Error())
			continue
		}

		if dryRun {
			dryRunCount++
			log.Infof("DRY RUN: would push schema %s to %s", slug, toWorkspace)
			continue
		}

		err = mgmntClient.PushSchema(toWorkspace, slug, sch, commit, commitMessage)
		if err != nil {
			errors = append(errors, fmt.Sprintf("schemas/%s: failed to push: %v", slug, err))
			log.Errorf("schemas/%s: failed to push: %v", slug, err)
			continue
		}

		successCount++
		log.Infof("Pushed schema: %s", slug)
	}
	if len(errors) > 0 {
		spinner.Stop(fmt.Sprintf("Synced schemas with %d error(s)", len(errors)))
		return clierr.New(fmt.Sprintf("one or more schemas failed to sync:\n%s", strings.Join(errors, "\n")), clierr.CodeAPIInternal)
	}
	if dryRun {
		spinner.Stop(fmt.Sprintf("DRY RUN: would push %d schema(s) to %s", dryRunCount, toWorkspace))
	} else {
		spinner.Stop(fmt.Sprintf("Synced %d schema(s) to %s", successCount, toWorkspace))
	}
	return nil
}

func syncEvents(mgmntClient *mgmnt.SS_MgmntClient, fromWorkspace, toWorkspace, dirPath string, dryRun bool) error {
	if dirPath == "" {
		dirPath = filepath.Join(".", "suprsend", "events")
	} else {
		dirPath = filepath.Join(dirPath, "events")
	}

	spinner := utils.NewSpinner(fmt.Sprintf("Pulling events from %s ...", fromWorkspace))

	events_resp, err := mgmntClient.GetEvents(fromWorkspace)
	if err != nil {
		spinner.Stop("")
		return clierr.Wrap(err, clierr.CodeAPIInternal, "error getting events")
	}
	_, err = event.WriteEventsToFiles(events_resp, dirPath)
	if err != nil {
		spinner.Stop("")
		return clierr.Wrap(err, clierr.CodeFileParseFailed, "error writing events to files")
	}
	events, err := event.ReadEventsFromDir(dirPath)
	if err != nil {
		spinner.Stop("")
		return clierr.Wrap(err, clierr.CodeFileParseFailed, "error reading events from files")
	}
	if dryRun {
		spinner.Stop(fmt.Sprintf("DRY RUN: would push %d event(s) to %s", len(events), toWorkspace))
		return nil
	}
	spinner.UpdateMessage(fmt.Sprintf("Pushing events to %s ...", toWorkspace))
	err = mgmntClient.PushEventsFromPayload(toWorkspace, map[string]any{"events": events})
	if err != nil {
		spinner.Stop("")
		return clierr.Wrap(err, clierr.CodeAPIInternal, "error pushing events")
	}
	spinner.Stop(fmt.Sprintf("Synced %d event(s) to %s", len(events), toWorkspace))
	return nil
}

func syncCategories(mgmntClient *mgmnt.SS_MgmntClient, fromWorkspace, toWorkspace, mode, dirPath string, commit bool, commitMessage string, dryRun bool) error {
	if dirPath == "" {
		dirPath = filepath.Join(".", "suprsend", "preference_categories")
	} else {
		dirPath = filepath.Join(dirPath, "preference_categories")
	}

	spinner := utils.NewSpinner(fmt.Sprintf("Pulling categories from %s ...", fromWorkspace))

	categoriesResp, err := mgmntClient.ListCategories(fromWorkspace, mode)
	if err != nil {
		spinner.Stop("")
		return clierr.Wrap(err, clierr.CodeAPIInternal, "error getting categories")
	}
	filePath := filepath.Join(dirPath, "categories.json")
	err = category.WriteToFile(categoriesResp, filePath)
	if err != nil {
		spinner.Stop("")
		return clierr.Wrap(err, clierr.CodeFileParseFailed, "error writing categories to files")
	}
	categories, err := category.ReadFromFile(filePath)
	if err != nil {
		spinner.Stop("")
		return clierr.Wrap(err, clierr.CodeFileParseFailed, "error reading categories from file")
	}
	if dryRun {
		spinner.Stop(fmt.Sprintf("DRY RUN: would push categories to %s", toWorkspace))
		return nil
	}

	spinner.UpdateMessage(fmt.Sprintf("Pushing categories to %s ...", toWorkspace))
	err = mgmntClient.PushCategories(toWorkspace, categories, commit, commitMessage)
	if err != nil {
		spinner.Stop("")
		return clierr.Wrap(err, clierr.CodeAPIInternal, "error pushing categories")
	}
	spinner.Stop(fmt.Sprintf("Pushed categories to %s", toWorkspace))

	// Sync category translations
	if err := syncCategoryTranslations(mgmntClient, fromWorkspace, toWorkspace, dirPath); err != nil {
		return clierr.Wrap(err, clierr.CodeAPIInternal, "error syncing category translations")
	}

	return nil
}

func syncCategoryTranslations(mgmntClient *mgmnt.SS_MgmntClient, fromWorkspace, toWorkspace, dirPath string) error {
	dirPath = filepath.Join(dirPath, "translations")
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		return clierr.Wrap(err, clierr.CodeFileNotFound, "failed to create translations directory")
	}

	spinner := utils.NewSpinner(fmt.Sprintf("Pulling category translations from %s ...", fromWorkspace))

	locales, err := mgmntClient.ListPreferenceTranslations(fromWorkspace)
	if err != nil {
		spinner.Stop("")
		return clierr.Wrap(err, clierr.CodeAPIInternal, "error getting preference translation locales")
	}

	var errors []string
	successCount := 0

	for _, localeResult := range locales.Results {
		locale := localeResult.Locale

		// Skip English translations
		if locale == "en" {
			continue
		}

		translations, err := mgmntClient.GetPreferenceTranslationsForLocale(fromWorkspace, locale)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to fetch translations for locale %s: %v", locale, err))
			log.Errorf("Failed to fetch translations for locale %s: %v", locale, err)
			continue
		}

		// Write translation file locally
		filename := filepath.Join(dirPath, fmt.Sprintf("%s.json", locale))
		fileData, err := json.MarshalIndent(translations, "", "  ")
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to serialize translations for locale %s: %v", locale, err))
			log.Errorf("Failed to serialize translations for locale %s: %v", locale, err)
			continue
		}

		if err := os.WriteFile(filename, fileData, 0644); err != nil {
			errors = append(errors, fmt.Sprintf("failed to write translation file for locale %s: %v", locale, err))
			log.Errorf("Failed to write translation file for locale %s: %v", locale, err)
			continue
		}

		// Push translation to destination workspace
		err = mgmntClient.PushPreferenceTranslation(toWorkspace, locale, *translations)
		if err != nil {
			errors = append(errors, fmt.Sprintf("preference_categories/translations/%s.json: failed to push: %v", locale, err))
			log.Errorf("preference_categories/translations/%s.json: failed to push: %v", locale, err)
			continue
		}

		successCount++
		log.Infof("Pushed category translations for locale: %s", locale)
	}

	if len(errors) > 0 {
		spinner.Stop(fmt.Sprintf("Synced category translations with %d error(s)", len(errors)))
		return clierr.New(fmt.Sprintf("one or more category translations failed to sync:\n%s", strings.Join(errors, "\n")), clierr.CodeAPIInternal)
	}

	if successCount > 0 {
		spinner.Stop(fmt.Sprintf("Synced %d category translation locale(s) to %s", successCount, toWorkspace))
	} else {
		spinner.Stop("No category translations to sync")
	}

	return nil
}

func syncTranslation(mgmntClient *mgmnt.SS_MgmntClient, fromWorkspace, toWorkspace, mode, dirPath string, commit bool, commitMessage string, dryRun bool) error {
	if dirPath == "" {
		dirPath = filepath.Join(".", "suprsend", "translations")
	} else {
		dirPath = filepath.Join(dirPath, "translations")
	}

	spinner := utils.NewSpinner(fmt.Sprintf("Pulling translations from %s ...", fromWorkspace))

	translations_resp, err := mgmntClient.GetTranslations(fromWorkspace, mode)
	if err != nil {
		spinner.Stop("")
		return clierr.Wrap(err, clierr.CodeAPIInternal, "error getting translations")
	}
	_, err = translation.WriteTranslationToFiles(*translations_resp, dirPath)
	if err != nil {
		spinner.Stop("")
		return clierr.Wrap(err, clierr.CodeFileParseFailed, "error writing translations to files")
	}
	files, err := os.ReadDir(dirPath)
	if err != nil {
		spinner.Stop("")
		return clierr.Wrap(err, clierr.CodeFileNotFound, "error reading local translations directory")
	}

	spinner.UpdateMessage(fmt.Sprintf("Pushing translations to %s ...", toWorkspace))

	var errors []string
	successCount := 0
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		path := filepath.Join(dirPath, file.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			errors = append(errors, fmt.Sprintf("error reading file %s: %v", file.Name(), err))
			continue
		}

		var translation map[string]any
		if err := json.Unmarshal(data, &translation); err != nil {
			errors = append(errors, fmt.Sprintf("error unmarshalling JSON for %s: %v", file.Name(), err))
			continue
		}

		if dryRun {
			log.Infof("DRY RUN: would push translation %s to %s", file.Name(), toWorkspace)
			continue
		}

		err = mgmntClient.PushTranslation(toWorkspace, file.Name(), map[string]any{"content": translation})
		if err != nil {
			errors = append(errors, fmt.Sprintf("translations/%s: failed to push: %v", file.Name(), err))
			log.Errorf("translations/%s: failed to push: %v", file.Name(), err)
			continue
		}

		successCount++
		log.Infof("Pushed translation: %s", file.Name())
	}
	if len(errors) > 0 {
		spinner.Stop(fmt.Sprintf("Synced translations with %d error(s)", len(errors)))
		return clierr.New(fmt.Sprintf("one or more translations failed to sync:\n%s", strings.Join(errors, "\n")), clierr.CodeAPIInternal)
	}

	if commit && !dryRun {
		spinner.UpdateMessage(fmt.Sprintf("Committing translations on %s ...", toWorkspace))
		if err := mgmntClient.FinalizeTranslation(toWorkspace, commitMessage); err != nil {
			spinner.Stop("")
			return clierr.Wrap(err, clierr.CodeAPIInternal, fmt.Sprintf("failed to commit translations on %s", toWorkspace))
		}
		spinner.Stop(fmt.Sprintf("Synced and committed %d translation(s) on %s", successCount, toWorkspace))
		return nil
	}

	if dryRun {
		spinner.Stop(fmt.Sprintf("DRY RUN: would push %d translation(s) to %s", successCount, toWorkspace))
	} else {
		spinner.Stop(fmt.Sprintf("Synced %d translation(s) to %s", successCount, toWorkspace))
	}
	return nil
}

func syncTemplates(mgmntClient *mgmnt.SS_MgmntClient, fromWorkspace, toWorkspace, mode, dirPath string, commit bool, commitMessage string, dryRun bool) error {
	if dirPath == "" {
		dirPath = filepath.Join(".", "suprsend", "templates")
	} else {
		dirPath = filepath.Join(dirPath, "templates")
	}

	spinner := utils.NewSpinner(fmt.Sprintf("Pulling templates from %s ...", fromWorkspace))

	results, err := template.FetchTemplates(mgmntClient, fromWorkspace, mode, "", template.FetchOptions{})
	if err != nil {
		spinner.Stop("")
		return clierr.Wrap(err, clierr.CodeAPIInternal, "error getting templates")
	}

	writeStats, err := template.WriteTemplatesToFiles(results, dirPath)
	if err != nil {
		spinner.Stop("")
		return clierr.Wrap(err, clierr.CodeFileParseFailed, "error writing templates to files")
	}
	log.Infof("Wrote %d templates locally (%d failed)", writeStats.Success, writeStats.Failed)

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		spinner.Stop("")
		return clierr.Wrap(err, clierr.CodeFileNotFound, "error reading local templates directory")
	}

	spinner.UpdateMessage(fmt.Sprintf("Pushing templates to %s ...", toWorkspace))

	pushStats := &template.TemplatePushStats{Errors: []string{}}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pushStats.Total++
		slug := entry.Name()
		templateDir := filepath.Join(dirPath, slug)
		if err := template.PushTemplate(mgmntClient, toWorkspace, slug, templateDir, commitMessage, commit, true, dryRun); err != nil {
			log.Errorf("templates/%s: failed to push: %v", slug, err)
			pushStats.Failed++
			pushStats.Errors = append(pushStats.Errors, fmt.Sprintf("templates/%s: failed to push: %v", slug, err))
		} else {
			pushStats.Success++
		}
	}

	var errors []string
	for _, e := range writeStats.Errors {
		errors = append(errors, fmt.Sprintf("local write: %s", e))
	}
	for _, e := range pushStats.Errors {
		errors = append(errors, fmt.Sprintf("push: %s", e))
	}
	if len(errors) > 0 {
		spinner.Stop(fmt.Sprintf("Synced templates with %d error(s)", len(errors)))
		return clierr.New(
			fmt.Sprintf("one or more templates failed to sync (%d local-write failures, %d push failures):\n%s",
				writeStats.Failed, pushStats.Failed, strings.Join(errors, "\n")),
			clierr.CodeAPIInternal,
		)
	}
	if dryRun {
		spinner.Stop(fmt.Sprintf("DRY RUN: would push %d template(s) to %s", pushStats.Total, toWorkspace))
	} else {
		spinner.Stop(fmt.Sprintf("Synced %d template(s) to %s", pushStats.Success, toWorkspace))
	}
	return nil
}
