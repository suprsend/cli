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
	"github.com/yarlson/pin"
)

var translationPushCmd = &cobra.Command{
	Use:   "push",
	Short: "push workflows from local to suprsend",
	Long:  "push workflows from local to suprsend",
	Run: func(cmd *cobra.Command, args []string) {
		workspace, _ := cmd.Flags().GetString("workspace")
		outputDir, _ := cmd.Flags().GetString("dir")
		commit, _ := cmd.Flags().GetString("commit")
		commitMessage, _ := cmd.Flags().GetString("commit-message")

		if outputDir == "" {
			outputDir = filepath.Join(".", "suprsend", "translation")
		}

		files, err := os.ReadDir(outputDir)
		if err != nil {
			log.WithError(err).Errorf("Failed to read local translation directory")
			return
		}

		fmt.Printf("Pushing translations to %s\n", workspace)
		mgmntClient := utils.GetSuprSendMgmntClient()

		hasError := false
		var p *pin.Pin
		var cancel context.CancelFunc
		stats := &TranslationPushStats{
			Errors: []string{},
		}

		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
				continue
			}

			if !hasError && !utils.IsOutputPiped() {
				p = pin.New(fmt.Sprintf("Pushing %s...", file.Name()),
					pin.WithSpinnerColor(pin.ColorCyan),
					pin.WithTextColor(pin.ColorYellow),
				)
				cancel = p.Start(context.Background())
			}

			stats.Total++
			path := filepath.Join(outputDir, file.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				if p != nil && cancel != nil {
					p.Stop("")
					cancel()
					p = nil
					cancel = nil
				}
				hasError = true
				log.Errorf("Failed to read file %s: %v", file.Name(), err)
				stats.Failed++
				stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to read file %s: %v", file.Name(), err))
				continue
			}

			var content map[string]any
			if err := json.Unmarshal(data, &content); err != nil {
				if p != nil && cancel != nil {
					p.Stop("")
					cancel()
					p = nil
					cancel = nil
				}
				hasError = true
				log.Errorf("Failed to parse JSON for %s: %v", file.Name(), err)
				stats.Failed++
				stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to parse JSON for %s: %v", file.Name(), err))
				continue
			}

			err = mgmntClient.PushTranslation(workspace, file.Name(), map[string]any{"content": content})
			if err != nil {
				if p != nil && cancel != nil {
					p.Stop("")
					cancel()
					p = nil
					cancel = nil
				}
				hasError = true
				log.Errorf("Failed to push translation %s: %v", file.Name(), err)
				stats.Failed++
				stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to push translation %s: %v", file.Name(), err))
				continue
			}

			stats.Success++
			if p != nil && cancel != nil {
				p.Stop(fmt.Sprintf("Pushed translation: %s", file.Name()))
				cancel()
				p = nil
				cancel = nil
			} else {
				fmt.Fprintf(os.Stdout, "Pushed translation: %s\n", file.Name())
			}
			hasError = false
		}

		fmt.Fprintf(os.Stdout, "\n=== Translation Push Summary ===\n")
		fmt.Fprintf(os.Stdout, "Total translations processed: %d\n", stats.Total)
		fmt.Fprintf(os.Stdout, "Successfully pushed: %d\n", stats.Success)
		fmt.Fprintf(os.Stdout, "Failed to push: %d\n", stats.Failed)

		if stats.Failed > 0 {
			fmt.Fprintf(os.Stdout, "\nFailed translations:\n")
			for _, err := range stats.Errors {
				fmt.Fprintf(os.Stdout, "  - %s\n", err)
			}
		}
		if commit == "true" {
			err := mgmntClient.FinalizeTranslation(workspace, commitMessage)
			if err != nil {
				log.Errorf("Failed to commit translation: %v", err)
			}
			fmt.Fprintf(os.Stdout, "Committed translation: %s\n", commitMessage)
		}
	},
}

func init() {
	translationPushCmd.Flags().StringP("commit", "c", "false", "Commit the translation (--commit=true)")
	translationPushCmd.Flags().StringP("commit-message", "m", "", "Commit message for the translation")
	translationPushCmd.Flags().StringP("dir", "d", "", "Directory for translations pull to (default: ./suprsend/translation)")
	TranslationCmd.AddCommand(translationPushCmd)
}
