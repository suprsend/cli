package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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
	Run: func(cmd *cobra.Command, args []string) {
		mode, _ := cmd.Flags().GetString("mode")
		fromWorkspace, _ := cmd.Flags().GetString("from")
		toWorkspace, _ := cmd.Flags().GetString("to")
		assets, _ := cmd.Flags().GetString("assets")
		dirPath, _ := cmd.Flags().GetString("dir")
		commit, _ := cmd.Flags().GetBool("commit")
		commitMessage, _ := cmd.Flags().GetString("commit-message")

		if commit && commitMessage == "" {
			log.Error("--commit-message is required when --commit is set")
			return
		}

		if fromWorkspace == toWorkspace {
			log.Error("Cannot sync within the same workspace. Source and destination workspaces must be different.")
			return
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
			log.Errorf("Invalid asset type: '%s'. Valid options are: all, workflow, schema, event, category, translation, template", assets)
			return
		}

		log.Infof("Syncing assets from %s to %s ...", fromWorkspace, toWorkspace)
		log.Infof("Assets to sync: %v", assetsToSync)

		mgmntClient := utils.GetSuprSendMgmntClient()
		hasErrors := false

		for _, assetType := range assetsToSync {
			switch assetType {
			case "workflow":
				err := syncWorkflows(mgmntClient, fromWorkspace, toWorkspace, mode, dirPath, commit, commitMessage)
				if err != nil {
					log.WithError(err).Errorf("Failed to sync workflows")
					hasErrors = true
				}
			case "schema":
				err := syncSchemas(mgmntClient, fromWorkspace, toWorkspace, mode, dirPath, commit, commitMessage)
				if err != nil {
					log.WithError(err).Errorf("Failed to sync schemas")
					hasErrors = true
				}
			case "event":
				err := syncEvents(mgmntClient, fromWorkspace, toWorkspace, dirPath)
				if err != nil {
					log.WithError(err).Errorf("Failed to sync events")
					hasErrors = true
				}
			case "category":
				err := syncCategories(mgmntClient, fromWorkspace, toWorkspace, mode, dirPath, commit, commitMessage)
				if err != nil {
					log.WithError(err).Errorf("Failed to sync categories")
					hasErrors = true
				}
			case "translation":
				err := syncTranslation(mgmntClient, fromWorkspace, toWorkspace, mode, dirPath, commit, commitMessage)
				if err != nil {
					log.WithError(err).Errorf("Failed to sync translations")
					hasErrors = true
				}
			case "template":
				err := syncTemplates(mgmntClient, fromWorkspace, toWorkspace, mode, dirPath, commit, commitMessage)
				if err != nil {
					log.WithError(err).Errorf("Failed to sync templates")
					hasErrors = true
				}
			default:
				log.Errorf("Invalid asset type: %s", assetType)
			}
		}
		if hasErrors {
			log.Error("Sync complete with errors")
		} else {
			log.Info("Sync complete")
		}
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)

	// Flags consumed in Run
	syncCmd.Flags().StringP("from", "f", "staging", "Source workspace to pull assets from")
	syncCmd.Flags().StringP("to", "t", "production", "Destination workspace to push assets to")
	syncCmd.Flags().StringP("dir", "d", "", "Local directory for intermediate file storage during sync")
	syncCmd.Flags().StringP("mode", "m", "live", "Version mode: draft or live")
	syncCmd.Flags().StringP("assets", "a", "all", "Asset types to sync: all, workflow, schema, event, category, translation, or template")
	syncCmd.Flags().BoolP("commit", "c", false, "Promote changes from draft to live after syncing")
	syncCmd.Flags().String("commit-message", "", "Commit message applied to every committed resource in this sync run (required when --commit is set)")
}

func syncWorkflows(mgmntClient *mgmnt.SS_MgmntClient, fromWorkspace, toWorkspace, mode, dirPath string, commit bool, commitMessage string) error {
	if dirPath == "" {
		dirPath = filepath.Join(".", "suprsend", "workflows")
	} else {
		dirPath = filepath.Join(dirPath, "workflows")
	}

	workflows_resp, err := mgmntClient.GetWorkflows(fromWorkspace, mode)
	if err != nil {
		return fmt.Errorf("error getting workflows: %w", err)
	}

	log.Infof("Pulling workflows from %s ... \n", fromWorkspace)
	_, err = workflow.WriteWorkflowsToFiles(*workflows_resp, dirPath)
	if err != nil {
		return fmt.Errorf("error writing workflows to files: %w", err)
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("error reading local workflows directory: %w", err)
	}

	var errors []string
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
		delete(wf, "$schema")

		err = mgmntClient.PushWorkflow(toWorkspace, slug, wf, commit, commitMessage)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to push workflow %s: %v", slug, err))
			log.WithError(err).Errorf("Failed to push workflow %s", slug)
			continue
		}

		log.Infof("Pushed workflow: %s\n", slug)
	}
	if len(errors) > 0 {
		return fmt.Errorf("one or more workflows failed to sync:\n%s", strings.Join(errors, "\n"))
	}
	return nil
}

func syncSchemas(mgmntClient *mgmnt.SS_MgmntClient, fromWorkspace, toWorkspace, mode, dirPath string, commit bool, commitMessage string) error {
	if dirPath == "" {
		dirPath = filepath.Join(".", "suprsend", "schemas")
	} else {
		dirPath = filepath.Join(dirPath, "schemas")
	}

	log.Infof("Pulling schemas from %s ...", fromWorkspace)
	schemas_resp, err := mgmntClient.GetSchemas(fromWorkspace, mode)
	if err != nil {
		return fmt.Errorf("error getting schemas: %w", err)
	}

	_, err = schema.WriteSchemasToFiles(schemas_resp, dirPath)
	if err != nil {
		return fmt.Errorf("error writing schemas to files: %w", err)
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("error reading local schemas directory: %w", err)
	}

	var errors []string
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

		err = mgmntClient.PushSchema(toWorkspace, slug, sch, commit, commitMessage)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to push schema %s: %v", slug, err))
			log.WithError(err).Errorf("Failed to push schema %s", slug)
			continue
		}

		log.Infof("Pushed schema: %s\n", slug)
	}
	if len(errors) > 0 {
		return fmt.Errorf("one or more schemas failed to sync:\n%s", strings.Join(errors, "\n"))
	}
	return nil
}

func syncEvents(mgmntClient *mgmnt.SS_MgmntClient, fromWorkspace, toWorkspace, dirPath string) error {
	if dirPath == "" {
		dirPath = filepath.Join(".", "suprsend", "events")
	} else {
		dirPath = filepath.Join(dirPath, "events")
	}

	log.Infof("Pulling events from %s ...", fromWorkspace)
	events_resp, err := mgmntClient.GetEvents(fromWorkspace)
	if err != nil {
		return fmt.Errorf("error getting events: %w", err)
	}
	_, err = event.WriteEventsToFiles(events_resp, dirPath)
	if err != nil {
		return fmt.Errorf("error writing events to files: %w", err)
	}
	events, err := event.ReadEventsFromDir(dirPath)
	if err != nil {
		return fmt.Errorf("error reading events from files: %w", err)
	}
	log.Infof("Pushing events to %s ...", toWorkspace)
	err = mgmntClient.PushEventsFromPayload(toWorkspace, map[string]any{"events": events})
	if err != nil {
		return fmt.Errorf("error pushing events: %w", err)
	}
	return nil
}

func syncCategories(mgmntClient *mgmnt.SS_MgmntClient, fromWorkspace, toWorkspace, mode, dirPath string, commit bool, commitMessage string) error {
	if dirPath == "" {
		dirPath = filepath.Join(".", "suprsend", "preference_categories")
	} else {
		dirPath = filepath.Join(dirPath, "preference_categories")
	}
	categoriesResp, err := mgmntClient.ListCategories(fromWorkspace, mode)
	if err != nil {
		return fmt.Errorf("error getting categories: %w", err)
	}
	log.Infof("Pulling categories from %s ...", fromWorkspace)
	filePath := filepath.Join(dirPath, "categories.json")
	err = category.WriteToFile(categoriesResp, filePath)
	if err != nil {
		return fmt.Errorf("error writing categories to files: %w", err)
	}
	categories, err := category.ReadFromFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading categories from file: %w", err)
	}
	err = mgmntClient.PushCategories(toWorkspace, categories, commit, commitMessage)
	if err != nil {
		return fmt.Errorf("error pushing categories: %w", err)
	}
	log.Printf("Pushed categories to %s", toWorkspace)

	// Sync category translations
	if err := syncCategoryTranslations(mgmntClient, fromWorkspace, toWorkspace, dirPath); err != nil {
		return fmt.Errorf("error syncing category translations: %w", err)
	}

	return nil
}

func syncCategoryTranslations(mgmntClient *mgmnt.SS_MgmntClient, fromWorkspace, toWorkspace, dirPath string) error {
	dirPath = filepath.Join(dirPath, "translations")
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		return fmt.Errorf("failed to create translations directory: %w", err)
	}
	log.Infof("Pulling category translations from %s ...", fromWorkspace)
	locales, err := mgmntClient.ListPreferenceTranslations(fromWorkspace)
	if err != nil {
		return fmt.Errorf("error getting preference translation locales: %w", err)
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
			log.WithError(err).Errorf("Failed to fetch translations for locale %s", locale)
			continue
		}

		// Write translation file locally
		filename := filepath.Join(dirPath, fmt.Sprintf("%s.json", locale))
		fileData, err := json.MarshalIndent(translations, "", "  ")
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to serialize translations for locale %s: %v", locale, err))
			log.WithError(err).Errorf("Failed to serialize translations for locale %s", locale)
			continue
		}

		if err := os.WriteFile(filename, fileData, 0644); err != nil {
			errors = append(errors, fmt.Sprintf("failed to write translation file for locale %s: %v", locale, err))
			log.WithError(err).Errorf("Failed to write translation file for locale %s", locale)
			continue
		}

		// Push translation to destination workspace
		err = mgmntClient.PushPreferenceTranslation(toWorkspace, locale, *translations)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to push translations for locale %s: %v", locale, err))
			log.WithError(err).Errorf("Failed to push translations for locale %s", locale)
			continue
		}

		successCount++
		log.Infof("Pushed category translations for locale: %s", locale)
	}

	if len(errors) > 0 {
		return fmt.Errorf("one or more category translations failed to sync:\n%s", strings.Join(errors, "\n"))
	}

	if successCount > 0 {
		log.Printf("Pushed %d category translation locale(s) to %s", successCount, toWorkspace)
	}

	return nil
}

func syncTranslation(mgmntClient *mgmnt.SS_MgmntClient, fromWorkspace, toWorkspace, mode, dirPath string, commit bool, commitMessage string) error {
	if dirPath == "" {
		dirPath = filepath.Join(".", "suprsend", "translations")
	} else {
		dirPath = filepath.Join(dirPath, "translations")
	}

	log.Infof("Pulling translations from %s ...", fromWorkspace)
	translations_resp, err := mgmntClient.GetTranslations(fromWorkspace, mode)
	if err != nil {
		return fmt.Errorf("error getting translations: %w", err)
	}
	_, err = translation.WriteTranslationToFiles(*translations_resp, dirPath)
	if err != nil {
		return fmt.Errorf("error writing translations to files: %w", err)
	}
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("error reading local translations directory: %w", err)
	}

	var errors []string
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

		err = mgmntClient.PushTranslation(toWorkspace, file.Name(), map[string]any{"content": translation})
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to push translation %s: %v", file.Name(), err))
			log.WithError(err).Errorf("Failed to push translation %s", file.Name())
			continue
		}

		log.Infof("Pushed translation: %s\n", file.Name())
	}
	if len(errors) > 0 {
		return fmt.Errorf("one or more translations failed to sync:\n%s", strings.Join(errors, "\n"))
	}

	if commit {
		if err := mgmntClient.FinalizeTranslation(toWorkspace, commitMessage); err != nil {
			return fmt.Errorf("failed to commit translations on %s: %w", toWorkspace, err)
		}
		log.Infof("Committed translations as live on %s", toWorkspace)
	}

	return nil
}

func syncTemplates(mgmntClient *mgmnt.SS_MgmntClient, fromWorkspace, toWorkspace, mode, dirPath string, commit bool, commitMessage string) error {
	if dirPath == "" {
		dirPath = filepath.Join(".", "suprsend", "templates")
	} else {
		dirPath = filepath.Join(dirPath, "templates")
	}

	log.Infof("Pulling templates from %s ...", fromWorkspace)
	results, err := template.FetchTemplates(mgmntClient, fromWorkspace, mode, "")
	if err != nil {
		return fmt.Errorf("error getting templates: %w", err)
	}

	writeStats, err := template.WriteTemplatesToFiles(results, dirPath)
	if err != nil {
		return fmt.Errorf("error writing templates to files: %w", err)
	}
	log.Infof("Wrote %d templates locally (%d failed)", writeStats.Success, writeStats.Failed)

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("error reading local templates directory: %w", err)
	}

	pushStats := &template.TemplatePushStats{Errors: []string{}}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pushStats.Total++
		slug := entry.Name()
		templateDir := filepath.Join(dirPath, slug)
		log.Infof("Pushing template %s to %s ...", slug, toWorkspace)
		template.PushTemplate(mgmntClient, toWorkspace, slug, templateDir, commitMessage, commit, true, pushStats)
	}

	if pushStats.Success > 0 {
		log.Printf("Pushed %d template(s) to %s", pushStats.Success, toWorkspace)
	}

	var errors []string
	for _, e := range writeStats.Errors {
		errors = append(errors, fmt.Sprintf("local write: %s", e))
	}
	for _, e := range pushStats.Errors {
		errors = append(errors, fmt.Sprintf("push: %s", e))
	}
	if len(errors) > 0 {
		return fmt.Errorf("one or more templates failed to sync (%d local-write failures, %d push failures):\n%s",
			writeStats.Failed, pushStats.Failed, strings.Join(errors, "\n"))
	}
	return nil
}
