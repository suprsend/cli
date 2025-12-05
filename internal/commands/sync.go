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
	"github.com/suprsend/cli/internal/commands/translation"
	"github.com/suprsend/cli/internal/commands/workflow"
	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/cli/mgmnt"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync SuprSend assets from one workspace to another",
	Long:  `Sync SuprSend assets from one workspace to another`,
	Run: func(cmd *cobra.Command, args []string) {
		mode, _ := cmd.Flags().GetString("mode")
		fromWorkspace, _ := cmd.Flags().GetString("from")
		toWorkspace, _ := cmd.Flags().GetString("to")
		assets, _ := cmd.Flags().GetString("assets")
		dirPath, _ := cmd.Flags().GetString("dir")
		if fromWorkspace == toWorkspace {
			log.Error("Cannot sync within the same workspace. Source and destination workspaces must be different.")
			return
		}

		var assetsToSync []string
		switch assets {
		case "all":
			assetsToSync = []string{"category", "schema", "event", "workflow", "translation"}
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
		default:
			log.Errorf("Invalid asset type: '%s'. Valid options are: all, workflow, schema, event, category, translation", assets)
			return
		}

		log.Infof("Syncing assets from %s to %s ...", fromWorkspace, toWorkspace)
		log.Infof("Assets to sync: %v", assetsToSync)

		mgmntClient := utils.GetSuprSendMgmntClient()
		hasErrors := false

		for _, assetType := range assetsToSync {
			switch assetType {
			case "workflow":
				err := syncWorkflows(mgmntClient, fromWorkspace, toWorkspace, mode, dirPath)
				if err != nil {
					log.WithError(err).Errorf("Failed to sync workflows")
					hasErrors = true
				}
			case "schema":
				err := syncSchemas(mgmntClient, fromWorkspace, toWorkspace, mode, dirPath)
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
				err := syncCategories(mgmntClient, fromWorkspace, toWorkspace, mode, dirPath)
				if err != nil {
					log.WithError(err).Errorf("Failed to sync categories")
					hasErrors = true
				}
			case "translation":
				err := syncTranslation(mgmntClient, fromWorkspace, toWorkspace, mode, dirPath)
				if err != nil {
					log.WithError(err).Errorf("Failed to sync translations")
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
	syncCmd.Flags().StringP("from", "f", "staging", "Source workspace (required)")
	syncCmd.Flags().StringP("to", "t", "production", "Destination workspace (required)")
	syncCmd.Flags().StringP("dir", "d", "", "Directory to sync assets to")
	syncCmd.Flags().StringP("mode", "m", "live", "Mode to sync assets (draft, live), default: live")
	syncCmd.Flags().StringP("assets", "a", "all", "Assets to sync (all, workflow, schema, event, category, translation)")
}

func syncWorkflows(mgmntClient *mgmnt.SS_MgmntClient, fromWorkspace, toWorkspace, mode, dirPath string) error {
	if dirPath == "" {
		dirPath = filepath.Join(".", "suprsend", "workflow")
	} else {
		dirPath = filepath.Join(dirPath, "workflow")
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

	files, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("error reading local workflows directory: %w", err)
	}

	var errors []string
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		slug := strings.TrimSuffix(file.Name(), ".json")
		path := filepath.Join(dirPath, file.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			errors = append(errors, fmt.Sprintf("error reading file %s: %v", file.Name(), err))
			continue
		}

		var wf map[string]any
		if err := json.Unmarshal(data, &wf); err != nil {
			errors = append(errors, fmt.Sprintf("error unmarshalling JSON for %s: %v", file.Name(), err))
			continue
		}

		err = mgmntClient.PushWorkflow(toWorkspace, slug, wf, "true", "")
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

func syncSchemas(mgmntClient *mgmnt.SS_MgmntClient, fromWorkspace, toWorkspace, mode, dirPath string) error {
	if dirPath == "" {
		dirPath = filepath.Join(".", "suprsend", "schema")
	} else {
		dirPath = filepath.Join(dirPath, "schema")
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

	files, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("error reading local schemas directory: %w", err)
	}

	var errors []string
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		slug := strings.TrimSuffix(file.Name(), ".json")
		path := filepath.Join(dirPath, file.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			errors = append(errors, fmt.Sprintf("error reading file %s: %v", file.Name(), err))
			continue
		}

		var sch map[string]any
		if err := json.Unmarshal(data, &sch); err != nil {
			errors = append(errors, fmt.Sprintf("error unmarshalling JSON for %s: %v", file.Name(), err))
			continue
		}

		err = mgmntClient.PushSchema(toWorkspace, slug, sch, "true", "")
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
		dirPath = filepath.Join(".", "suprsend", "event")
	} else {
		dirPath = filepath.Join(dirPath, "event")
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
	filePath := filepath.Join(dirPath, "event_schema_mapping.json")
	log.Infof("Pushing events to %s ...", toWorkspace)
	err = mgmntClient.PushEvents(toWorkspace, filePath)
	if err != nil {
		return fmt.Errorf("error pushing events: %w", err)
	}
	return nil
}

func syncCategories(mgmntClient *mgmnt.SS_MgmntClient, fromWorkspace, toWorkspace, mode, dirPath string) error {
	if dirPath == "" {
		dirPath = filepath.Join(".", "suprsend", "category")
	} else {
		dirPath = filepath.Join(dirPath, "category")
	}
	categoriesResp, err := mgmntClient.ListCategories(fromWorkspace, mode)
	if err != nil {
		return fmt.Errorf("error getting categories: %w", err)
	}
	log.Infof("Pulling categories from %s ...", fromWorkspace)
	filePath := filepath.Join(dirPath, "categories_preferences.json")
	err = category.WriteToFile(categoriesResp, filePath)
	if err != nil {
		return fmt.Errorf("error writing categories to files: %w", err)
	}
	categories, err := category.ReadFromFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading categories from file: %w", err)
	}
	err = mgmntClient.PushCategories(toWorkspace, categories, "true", "")
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

func syncTranslation(mgmntClient *mgmnt.SS_MgmntClient, fromWorkspace, toWorkspace, mode, dirPath string) error {
	if dirPath == "" {
		dirPath = filepath.Join(".", "suprsend", "translation")
	} else {
		dirPath = filepath.Join(dirPath, "translation")
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

		err = mgmntClient.PushTranslation(toWorkspace, file.Name(), translation)
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
	return nil
}
