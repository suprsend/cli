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
	"github.com/suprsend/cli/internal/state"
	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/cli/mgmnt"
)

var translationPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push translation files to a workspace",
	Long:  "Upload local template translation JSON files to a workspace. Reads all .json files from the input directory and pushes them. Use --commit to also finalize the changes immediately.",
	Example: `  # Push translations from default directory
  suprsend translation push

  # Push and commit to live immediately
  suprsend translation push --commit

  # Dry run: preview what would be pushed without making changes
  suprsend translation push --dry-run`,
	Annotations: map[string]string{
		"skills:tip.a-result": "Push first lists the workspace's draft translation files and sends only the files whose content differs. Each file is reported on stdout as `created`, `updated`, `unchanged` (not sent) or `failed`; with `--commit`, a final `translations` line says `committed` or `unchanged`. With `-o json` the result is one JSON object with `items` and `counts`.",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		outputDir, _ := cmd.Flags().GetString("dir")
		commit, _ := cmd.Flags().GetBool("commit")
		commitMessage, _ := cmd.Flags().GetString("commit-message")
		jsonPayload, _ := cmd.Flags().GetString("json")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		force, _ := cmd.Flags().GetBool("force")

		if commit && !dryRun && !force {
			msg := fmt.Sprintf("This will push and promote translations to live in workspace \"%s\". Continue?", workspace)
			confirmed, err := utils.ConfirmDestructiveAction(msg)
			if err != nil || !confirmed {
				log.Info("Aborted.")
				return nil
			}
		}

		mgmntClient := utils.GetSuprSendMgmntClient()
		report := utils.NewReport("translation push", workspace, dryRun)
		var failures []string
		var failedErrs []error
		fail := func(resource, action string, err error) {
			report.Fail(resource, err)
			failures = append(failures, clierr.ItemFailure(resource, action, err))
			failedErrs = append(failedErrs, err)
		}
		// The workspace's draft files, to send only the files that differ.
		var remote map[string]string
		if !dryRun {
			r, err := draftFingerprints(mgmntClient, workspace)
			if err != nil {
				return report.Finish(err)
			}
			remote = r
		}
		push := func(fileName string, content map[string]any) {
			resource := "translations/" + fileName
			if dryRun {
				report.Add(resource, utils.StatusWouldPush, "")
				return
			}
			fp, exists := remote[fileName]
			if exists && fp == state.Fingerprint(content) {
				report.Add(resource, utils.StatusUnchanged, "")
				return
			}
			spinner := utils.NewSpinner(fmt.Sprintf("Pushing %s...", fileName))
			err := mgmntClient.PushTranslation(workspace, fileName, map[string]any{"content": content})
			spinner.Stop("")
			if err != nil {
				fail(resource, "failed to push", err)
				return
			}
			status := utils.StatusUpdated
			if !exists {
				status = utils.StatusCreated
			}
			report.Add(resource, status, "")
		}

		if jsonPayload != "" {
			// Parse as map of filename -> content
			var translations map[string]map[string]any
			if err := json.Unmarshal([]byte(jsonPayload), &translations); err != nil {
				return clierr.Wrap(err, clierr.CodeFileParseFailed, "")
			}
			for filename, content := range translations {
				push(filename+".json", content)
			}
		} else {
			if outputDir == "" {
				outputDir = filepath.Join(".", "suprsend", "translations")
			}

			files, err := utils.ReadDir(outputDir)
			if err != nil {
				return clierr.Wrap(err, clierr.CodeFileNotFound, "")
			}

			for _, file := range files {
				if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
					continue
				}
				resource := "translations/" + file.Name()
				data, err := os.ReadFile(filepath.Join(outputDir, file.Name()))
				if err != nil {
					fail(resource, "failed to read", clierr.Wrap(err, clierr.CodeFileNotFound, ""))
					continue
				}
				var content map[string]any
				if err := json.Unmarshal(data, &content); err != nil {
					fail(resource, "failed to parse", clierr.Wrap(err, clierr.CodeFileParseFailed, ""))
					continue
				}
				push(file.Name(), content)
			}
		}

		if dryRun {
			if commit {
				report.Add("translations", utils.StatusWouldCommit, "all pushed translations")
			}
			if len(failures) > 0 {
				return report.Finish(clierr.Failures(fmt.Sprintf("%d translation(s) failed local checks", len(failures)), failures, clierr.CodeFileParseFailed))
			}
			return report.Finish(nil)
		}

		if commit {
			spinner := utils.NewSpinner("Committing translations...")
			commitStatus, err := mgmntClient.FinalizeTranslation(workspace, commitMessage)
			spinner.Stop("")
			if err != nil {
				err = clierr.Wrap(err, clierr.CodeAPIInternal, "")
				report.Fail("translations", err)
				return report.Finish(err)
			}
			status, detail := utils.PushStatus(commitStatus)
			report.Add("translations", status, detail)
		}

		if len(failures) > 0 {
			code := clierr.FailuresCode(failedErrs)
			summary := fmt.Sprintf("%d translation(s) failed to push", len(failures))
			if code == clierr.CodeFileParseFailed {
				summary = fmt.Sprintf("%d translation(s) failed local checks", len(failures))
			}
			return report.Finish(clierr.Failures(summary, failures, code))
		}
		return report.Finish(nil)
	},
}

func init() {
	translationPushCmd.Flags().BoolP("commit", "c", false, "Promote changes from draft to live after pushing")
	translationPushCmd.Flags().String("commit-message", "", "Message describing the changes being committed")
	translationPushCmd.Flags().StringP("dir", "d", "", "Directory containing translation JSON files (default: ./suprsend/translations)")
	translationPushCmd.Flags().StringP("json", "j", "", `Translations as a JSON object mapping locale codes (without .json extension) to their translation content objects, e.g. '{"en":{"key":"value"},"fr":{"key":"valeur"}}'`)
	translationPushCmd.Flags().BoolP("dry-run", "n", false, "Report what would be pushed without making any changes")
	translationPushCmd.Flags().BoolP("force", "F", false, "Skip confirmation prompt when --commit is set")
	TranslationCmd.AddCommand(translationPushCmd)
}

// draftFingerprints fingerprints the content of each of the workspace's
// draft translation files, by file name.
func draftFingerprints(client *mgmnt.SS_MgmntClient, workspace string) (map[string]string, error) {
	resp, err := client.GetTranslations(workspace, "draft")
	if err != nil {
		return nil, clierr.Wrap(err, clierr.CodeAPIInternal, "couldn't list the workspace's translations")
	}
	fps := map[string]string{}
	for _, r := range resp.Results {
		obj, ok := r.(map[string]any)
		if !ok {
			continue
		}
		// A file deleted in the draft is still listed, marked "deleted". It
		// must count as missing: an equal local file would otherwise be
		// "unchanged", and a commit would then delete it from live.
		if action, _ := obj["action"].(string); action == "deleted" {
			continue
		}
		if name, _ := obj["filename"].(string); name != "" {
			fps[name] = state.Fingerprint(obj["content"])
		}
	}
	return fps, nil
}
