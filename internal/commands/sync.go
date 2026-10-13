package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

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
	Long: "Sync notification assets from one workspace to another. For each asset type, sync pulls the source workspace (`--from`, default `staging`, in `--mode`, default `live`) into a local directory (`./suprsend/<type-dir>/`, or `<dir>/<type-dir>/` with `--dir`, where `<type-dir>` is `workflows`, `schemas`, `events`, `templates`, `translations` or `preference_categories`), then pushes every asset found in that directory to the destination workspace (`--to`, default `production`) and commits it, so synced assets go live in the destination immediately. " +
		"Supports syncing all asset types or a specific type (workflow, schema, event, category, translation, template). Source and destination workspaces must be different.",
	Example: `  # Sync all assets from staging to production (pushes and commits to live)
  suprsend sync --from staging --to production

  # Sync only workflows
  suprsend sync --from staging --to production --assets workflow

  # Dry run: pull staging into local files and list what would be pushed (destination untouched)
  suprsend sync --from staging --to production --dry-run

  # Sync through a fresh directory so only assets that exist in staging are pushed
  suprsend sync --from staging --to production --dir ./sync-tmp`,
	Annotations: map[string]string{
		"skills:tip.a-direction": "`--from` is the source (default `staging`), `--to` is the destination (default `production`); they must be different workspaces. Sync pushes **and commits** every asset type, so changes go **live** in the destination immediately. There is no draft-only sync.",
		"skills:tip.b-commit":    "Committing is per asset: a workflow or schema that is saved as a draft but not committed (for example a workflow that fails validation) counts as a failure, and sync exits non-zero after trying the rest. Template variants that fail pre-commit validation are skipped while the valid ones are committed; the skips are only logged as warnings, and a template with no valid changed variant counts as a failure. Events the destination rejects (typically a `payload_schema` pinned to a schema `version_no` that does not exist there, since each workspace numbers schema versions separately) are listed and count as a failure; the other events are still saved. Assets with nothing to commit (the destination already matches) count as success, so re-running sync is safe.",
		"skills:tip.c-local":     "Sync pushes **every** asset in the local sync directory, not just what it pulled: local-only workflows, schemas, events, templates or translation files under `./suprsend/{workflows,schemas,events,templates,translations}/` are pushed and committed too. Categories are the exception: `categories.json` is overwritten from the source. Pulled assets overwrite local files with the same slug. Use a fresh `--dir` to sync exactly what exists in the source.",
		"skills:tip.d-dryrun":    "`--dry-run` makes no calls to the destination, but it still pulls the source and **overwrites local files** in the sync directory (template slug directories are deleted and rewritten). Only local checks run (JSON parsing, plus a JSON Schema compile for schemas); nothing is validated server-side. Add `--assets <type>` to scope to one resource type (workflow / schema / event / category / translation / template).",
		"skills:tip.e-confirm":   "The confirmation prompt only appears when stdin is a TTY. In non-interactive shells (CI, agents) sync starts immediately, even without `--force`.",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		mode, _ := cmd.Flags().GetString("mode")
		fromWorkspace, _ := cmd.Flags().GetString("from")
		toWorkspace, _ := cmd.Flags().GetString("to")
		assets, _ := cmd.Flags().GetString("assets")
		dirPath, _ := cmd.Flags().GetString("dir")
		commit := true
		commitMessage, _ := cmd.Flags().GetString("commit-message")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		force, _ := cmd.Flags().GetBool("force")

		if fromWorkspace == toWorkspace {
			return clierr.New("cannot sync within the same workspace; source and destination workspaces must be different", clierr.CodeInvalidUsage)
		}
		if mode != "live" && mode != "draft" {
			return clierr.New(fmt.Sprintf("invalid mode '%s'; valid options are: draft, live", mode), clierr.CodeInvalidUsage)
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

		if !dryRun && !force {
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
		var failures []string

		for _, assetType := range assetsToSync {
			switch assetType {
			case "workflow":
				err := syncWorkflows(mgmntClient, fromWorkspace, toWorkspace, mode, dirPath, commit, commitMessage, dryRun)
				if err != nil {
					log.Errorf("Failed to sync workflows: %v", err)
					failures = append(failures, fmt.Sprintf("Failed to sync workflows: %v", err))
				}
			case "schema":
				err := syncSchemas(mgmntClient, fromWorkspace, toWorkspace, mode, dirPath, commit, commitMessage, dryRun)
				if err != nil {
					log.Errorf("Failed to sync schemas: %v", err)
					failures = append(failures, fmt.Sprintf("Failed to sync schemas: %v", err))
				}
			case "event":
				err := syncEvents(mgmntClient, fromWorkspace, toWorkspace, dirPath, dryRun)
				if err != nil {
					log.Errorf("Failed to sync events: %v", err)
					failures = append(failures, fmt.Sprintf("Failed to sync events: %v", err))
				}
			case "category":
				err := syncCategories(mgmntClient, fromWorkspace, toWorkspace, mode, dirPath, commit, commitMessage, dryRun)
				if err != nil {
					log.Errorf("Failed to sync categories: %v", err)
					failures = append(failures, fmt.Sprintf("Failed to sync categories: %v", err))
				}
			case "translation":
				err := syncTranslation(mgmntClient, fromWorkspace, toWorkspace, mode, dirPath, commit, commitMessage, dryRun)
				if err != nil {
					log.Errorf("Failed to sync translations: %v", err)
					failures = append(failures, fmt.Sprintf("Failed to sync translations: %v", err))
				}
			case "template":
				err := syncTemplates(mgmntClient, fromWorkspace, toWorkspace, mode, dirPath, commit, commitMessage, dryRun)
				if err != nil {
					log.Errorf("Failed to sync templates: %v", err)
					failures = append(failures, fmt.Sprintf("Failed to sync templates: %v", err))
				}
			}
		}
		if len(failures) > 0 {
			return clierr.Failures("sync complete with errors", failures, clierr.CodeAPIInternal)
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
	syncCmd.Flags().StringP("dir", "d", "", "Local directory for intermediate file storage; each asset type uses a subdirectory (default: ./suprsend)")
	syncCmd.Flags().StringP("mode", "m", "live", "Version of the source assets to pull: draft or live")
	syncCmd.Flags().StringP("assets", "a", "all", "Asset types to sync: all, workflow, schema, event, category, translation, or template")
	syncCmd.Flags().String("commit-message", "", "Commit message applied to every committed resource in this sync run")
	syncCmd.Flags().BoolP("dry-run", "n", false, "Pull the source into local files, then only print what would be pushed (no changes to the destination)")
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

	writeStats, err := workflow.WriteWorkflowsToFiles(*workflows_resp, dirPath)
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
	// Items that couldn't be written locally are never pushed; report them too.
	for _, e := range writeStats.Errors {
		errors = append(errors, "pull: "+e)
	}
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
			errors = append(errors, clierr.ItemFailure("workflows/"+slug, "failed to push", err))
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

	writeStats, err := schema.WriteSchemasToFiles(schemas_resp, dirPath)
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
	// Items that couldn't be written locally are never pushed; report them too.
	for _, e := range writeStats.Errors {
		errors = append(errors, "pull: "+e)
	}
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
			errors = append(errors, clierr.ItemFailure("schemas/"+slug, "failed to push", err))
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
	writeStats, err := event.WriteEventsToFiles(events_resp, dirPath)
	if err != nil {
		spinner.Stop("")
		return clierr.Wrap(err, clierr.CodeFileParseFailed, "error writing events to files")
	}
	// Events that couldn't be written locally are never pushed; report them too.
	var failures []string
	for _, e := range writeStats.Errors {
		failures = append(failures, "pull: "+e)
	}
	failed := func() error {
		if len(failures) == 0 {
			return nil
		}
		return clierr.Failures("one or more events failed to sync", failures, clierr.CodeAPIInternal)
	}

	events, err := event.ReadEventsFromDir(dirPath)
	if err != nil {
		spinner.Stop("")
		return clierr.Wrap(err, clierr.CodeFileParseFailed, "error reading events from files")
	}
	if len(events) == 0 {
		spinner.Stop("No events to sync")
		return failed()
	}
	if dryRun {
		spinner.Stop(fmt.Sprintf("DRY RUN: would push %d event(s) to %s", len(events), toWorkspace))
		return failed()
	}
	spinner.UpdateMessage(fmt.Sprintf("Pushing events to %s ...", toWorkspace))
	records, err := mgmntClient.PushEventsFromPayload(toWorkspace, map[string]any{"events": events})
	if err != nil {
		spinner.Stop("")
		return clierr.Wrap(err, clierr.CodeAPIInternal, "error pushing events")
	}
	if err := event.PushFailures(events, records); err != nil {
		failures = append(failures, err.Error())
	}
	if len(failures) > 0 {
		spinner.Stop("")
		return failed()
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
			errors = append(errors, clierr.ItemFailure("preference_categories/translations/"+locale+".json", "failed to push", err))
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
	writeStats, err := translation.WriteTranslationToFiles(*translations_resp, dirPath)
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
	// Translations that couldn't be written locally are never pushed; report them too.
	for _, e := range writeStats.Errors {
		errors = append(errors, "pull: "+e)
	}
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
			successCount++
			continue
		}

		err = mgmntClient.PushTranslation(toWorkspace, file.Name(), map[string]any{"content": translation})
		if err != nil {
			errors = append(errors, clierr.ItemFailure("translations/"+file.Name(), "failed to push", err))
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

	if successCount == 0 {
		// Nothing pushed, so nothing to commit (and a destination with no
		// translations yet has no draft to commit).
		spinner.Stop("No translations to sync")
		return nil
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

	var fetchMu sync.Mutex
	var fetchErrors []string
	results, err := template.FetchTemplates(mgmntClient, fromWorkspace, mode, "", template.FetchOptions{
		OnFetchError: func(slug string, err error) {
			fetchMu.Lock()
			fetchErrors = append(fetchErrors, clierr.ItemFailure("templates/"+slug, "couldn't fetch variants", err))
			fetchMu.Unlock()
		},
	})
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
			pushStats.Errors = append(pushStats.Errors, clierr.ItemFailure("templates/"+slug, "failed to push", err))
		} else {
			pushStats.Success++
		}
	}

	var errors []string
	for _, e := range fetchErrors {
		errors = append(errors, "pull: "+e)
	}
	for _, e := range writeStats.Errors {
		errors = append(errors, fmt.Sprintf("local write: %s", e))
	}
	for _, e := range pushStats.Errors {
		errors = append(errors, fmt.Sprintf("push: %s", e))
	}
	if len(errors) > 0 {
		spinner.Stop(fmt.Sprintf("Synced templates with %d error(s)", len(errors)))
		return clierr.New(
			fmt.Sprintf("one or more templates failed to sync (%d fetch failures, %d local-write failures, %d push failures):\n%s",
				len(fetchErrors), writeStats.Failed, pushStats.Failed, strings.Join(errors, "\n")),
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
