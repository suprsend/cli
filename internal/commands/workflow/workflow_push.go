package workflow

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

var workflowPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push workflows from local to SuprSend workspace",
	Long:  `Push workflows from local to SuprSend workspace`,
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		path, _ := cmd.Flags().GetString("dir")
		commit, _ := cmd.Flags().GetString("commit")
		commitMessage, _ := cmd.Flags().GetString("commit-message")
		slug, _ := cmd.Flags().GetString("slug")

		if path == "" {
			path = filepath.Join(".", "suprsend", "workflow")
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			log.Errorf("Directory %s does not exist", path)
			return err
		}

		if err := validateInputDirectory(path); err != nil {
			log.Errorf("Error with input directory: %v\n", err)
			return err
		}

		files, err := os.ReadDir(path)
		if err != nil {
			log.WithError(err).Errorf("Failed to read local workflows directory")
			return err
		}

		mgmntClient := utils.GetSuprSendMgmntClient()

		stats := &WorkflowPushStats{
			Errors: []string{},
		}

		hasError := false
		var p *pin.Pin
		var cancel context.CancelFunc

		if slug != "" {
			stats.Total = 1
			fileName := fmt.Sprintf("%s.json", slug)
			filePath := filepath.Join(path, fileName)

			if _, err := os.Stat(filePath); err != nil {
				log.WithError(err).Errorf("Failed to find workflow file %s", filePath)
				stats.Failed++
				stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to find workflow file %s: %v", filePath, err))
			} else {
				data, err := os.ReadFile(filePath)
				if err != nil {
					log.WithError(err).Errorf("Failed to read workflow file %s", filePath)
					stats.Failed++
					stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to read workflow file %s: %v", filePath, err))
				} else {
					if !hasError && !utils.IsOutputPiped() {
						p = pin.New(fmt.Sprintf("Pushing %s...", slug),
							pin.WithSpinnerColor(pin.ColorCyan),
							pin.WithTextColor(pin.ColorYellow),
						)
						cancel = p.Start(context.Background())
					}
					var workflow map[string]any
					if err := json.Unmarshal(data, &workflow); err != nil {
						log.WithError(err).Errorf("Failed to parse JSON for %s", filePath)
						stats.Failed++
						stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to parse JSON for %s: %v", filePath, err))
					} else {
						err = mgmntClient.PushWorkflow(workspace, slug, workflow, commit, commitMessage)
						if err != nil {
							log.WithError(err).Errorf("Failed to push workflow %s", slug)
							stats.Failed++
							stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to push workflow %s: %v", slug, err))
						} else {
							stats.Success++
						}
					}

					if p != nil && cancel != nil {
						if stats.Success > 0 {
							p.Stop(fmt.Sprintf("Pushed workflow: %s", slug))
						} else {
							p.Stop("")
						}
						cancel()
					} else if stats.Success > 0 {
						fmt.Fprintf(os.Stdout, "Pushed workflow: %s\n", slug)
					}
				}
			}

			fmt.Fprintf(os.Stdout, "\n=== Workflow Push Summary ===\n")
			fmt.Fprintf(os.Stdout, "Total workflows processed: %d\n", stats.Total)
			fmt.Fprintf(os.Stdout, "Successfully pushed: %d\n", stats.Success)
			fmt.Fprintf(os.Stdout, "Failed to push: %d\n", stats.Failed)

			if stats.Failed > 0 {
				fmt.Fprintf(os.Stdout, "\nFailed workflows:\n")
				for _, errorMsg := range stats.Errors {
					fmt.Fprintf(os.Stdout, "  - %s\n", errorMsg)
				}
				return fmt.Errorf("%d workflow(s) failed to push", stats.Failed)
			}
			return nil
		}
		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
				stats.Total++
			}
		}

		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
				continue
			}

			slug := strings.TrimSuffix(file.Name(), ".json")
			if !hasError && !utils.IsOutputPiped() {
				p = pin.New(fmt.Sprintf("Pushing %s...", slug),
					pin.WithSpinnerColor(pin.ColorCyan),
					pin.WithTextColor(pin.ColorYellow),
				)
				cancel = p.Start(context.Background())
			}
			filePath := filepath.Join(path, file.Name())
			data, err := os.ReadFile(filePath)
			if err != nil {
				if p != nil && cancel != nil {
					p.Stop("")
					cancel()
					p = nil
					cancel = nil
				}
				hasError = true
				log.WithError(err).Errorf("Failed to read file %s", file.Name())
				stats.Failed++
				stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to read file %s: %v", file.Name(), err))
				continue
			}

			var workflow map[string]any
			if err := json.Unmarshal(data, &workflow); err != nil {
				if p != nil && cancel != nil {
					p.Stop("")
					cancel()
					p = nil
					cancel = nil
				}
				hasError = true
				log.WithError(err).Errorf("Failed to parse JSON for %s", file.Name())
				stats.Failed++
				stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to parse JSON for %s: %v", file.Name(), err))
				continue
			}

			err = mgmntClient.PushWorkflow(workspace, slug, workflow, commit, commitMessage)
			if err != nil {
				if p != nil && cancel != nil {
					p.Stop("")
					cancel()
					p = nil
					cancel = nil
				}
				hasError = true
				log.WithError(err).Errorf("Failed to push workflow %s", slug)
				stats.Failed++
				stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to push workflow %s: %v", slug, err))
				continue
			}

			stats.Success++
			if p != nil && cancel != nil {
				p.Stop(fmt.Sprintf("Pushed workflow: %s", slug))
				cancel()
				p = nil
				cancel = nil
			} else {
				fmt.Fprintf(os.Stdout, "Pushed workflow: %s\n", slug)
			}
			hasError = false
		}

		fmt.Fprintf(os.Stdout, "\n=== Workflow Push Summary ===\n")
		fmt.Fprintf(os.Stdout, "Total workflows processed: %d\n", stats.Total)
		fmt.Fprintf(os.Stdout, "Successfully pushed: %d\n", stats.Success)
		fmt.Fprintf(os.Stdout, "Failed to push: %d\n", stats.Failed)

		if stats.Failed > 0 {
			fmt.Fprintf(os.Stdout, "\nFailed workflows:\n")
			for _, errorMsg := range stats.Errors {
				fmt.Fprintf(os.Stdout, "  - %s\n", errorMsg)
			}
			return fmt.Errorf("%d workflow(s) failed to push", stats.Failed)
		}
		return nil
	},
}

func init() {
	workflowPushCmd.PersistentFlags().StringP("dir", "d", "", "Output directory for workflows (default: ./suprsend/workflow)")
	workflowPushCmd.PersistentFlags().StringP("commit", "c", "true", "Commit the workflows (--commit=true)")
	workflowPushCmd.PersistentFlags().StringP("commit-message", "m", "", "Commit message describing the changes for --commit=true")
	workflowPushCmd.PersistentFlags().StringP("slug", "g", "", "Slug of the workflow to push")
	WorkflowCmd.AddCommand(workflowPushCmd)
}
