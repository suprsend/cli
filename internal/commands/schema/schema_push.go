package schema

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/utils"
	"github.com/yarlson/pin"
)

var schemaPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push schemas",
	Long:  "Upload local schema JSON files to a workspace. Reads .json files from the input directory and pushes them. By default, changes are staged as drafts. Use --commit to also promote to live. Use --slug to push a single schema.",
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		slug, _ := cmd.Flags().GetString("slug")
		commit, _ := cmd.Flags().GetBool("commit")
		commitMessage, _ := cmd.Flags().GetString("commit-message")
		path, _ := cmd.Flags().GetString("dir")
		jsonPayload, _ := cmd.Flags().GetString("json")

		if jsonPayload != "" && slug == "" {
			return fmt.Errorf("--json requires --slug to be specified")
		}

		mgmntClient := utils.GetSuprSendMgmntClient()

		stats := &SchemaPushStats{
			Errors: []string{},
		}

		hasError := false
		var p *pin.Pin
		var cancel context.CancelFunc

		if slug != "" {
			stats.Total = 1

			var schema map[string]any
			if jsonPayload != "" {
				if err := json.Unmarshal([]byte(jsonPayload), &schema); err != nil {
					return fmt.Errorf("failed to parse --json payload: %w", err)
				}
			} else {
				if path == "" {
					path = filepath.Join(".", "suprsend", "schemas")
				}
				if _, err := os.Stat(path); os.IsNotExist(err) {
					log.Errorf("Directory %s does not exist", path)
					return err
				}
				if err := validateInputDirectory(path); err != nil {
					log.Errorf("Error with input directory: %v\n", err)
					return err
				}

				merged, err := ReadAndMergeSchemaFiles(filepath.Join(path, slug), slug)
				if err != nil {
					log.WithError(err).Errorf("Failed to read schema files for %s", slug)
					stats.Failed++
					stats.Errors = append(stats.Errors, err.Error())
				} else {
					schema = merged
				}
			}

			if schema != nil {
				if !utils.IsOutputPiped() {
					p = pin.New(fmt.Sprintf("Pushing %s...", slug),
						pin.WithSpinnerColor(pin.ColorCyan),
						pin.WithTextColor(pin.ColorYellow),
					)
					cancel = p.Start(context.Background())
				}
				err := mgmntClient.PushSchema(workspace, slug, schema, commit, commitMessage)
				if p != nil && cancel != nil {
					if err == nil {
						p.Stop(fmt.Sprintf("Pushed schema: %s", slug))
					} else {
						p.Stop("")
					}
					cancel()
				} else if err == nil {
					fmt.Fprintf(os.Stdout, "Pushed schema: %s\n", slug)
				}
				if err != nil {
					log.WithError(err).Errorf("Failed to push schema %s", slug)
					stats.Failed++
					stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to push schema %s: %v", slug, err))
				} else {
					stats.Success++
				}
			}

			fmt.Fprintf(os.Stdout, "\n=== Schema Push Summary ===\n")
			fmt.Fprintf(os.Stdout, "Total schemas processed: %d\n", stats.Total)
			fmt.Fprintf(os.Stdout, "Successfully pushed: %d\n", stats.Success)
			fmt.Fprintf(os.Stdout, "Failed to push: %d\n", stats.Failed)

			if stats.Failed > 0 {
				fmt.Fprintf(os.Stdout, "\nFailed schemas:\n")
				for _, errorMsg := range stats.Errors {
					fmt.Fprintf(os.Stdout, "  - %s\n", errorMsg)
				}
				return fmt.Errorf("%d schema(s) failed to push", stats.Failed)
			}
			return nil
		}

		if path == "" {
			path = filepath.Join(".", "suprsend", "schemas")
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			log.Errorf("Directory %s does not exist", path)
			return err
		}

		if err := validateInputDirectory(path); err != nil {
			log.Errorf("Error with input directory: %v\n", err)
			return err
		}

		entries, err := os.ReadDir(path)
		if err != nil {
			log.WithError(err).Errorf("Failed to read local schema directory")
			return err
		}

		for _, entry := range entries {
			if entry.IsDir() {
				stats.Total++
			}
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			slug := entry.Name()
			if !hasError && !utils.IsOutputPiped() {
				p = pin.New(fmt.Sprintf("Pushing %s...", slug),
					pin.WithSpinnerColor(pin.ColorCyan),
					pin.WithTextColor(pin.ColorYellow),
				)
				cancel = p.Start(context.Background())
			}

			schema, err := ReadAndMergeSchemaFiles(filepath.Join(path, slug), slug)
			if err != nil {
				if p != nil && cancel != nil {
					p.Stop("")
					cancel()
					p = nil
					cancel = nil
				}
				hasError = true
				log.WithError(err).Errorf("Failed to read schema files for %s", slug)
				stats.Failed++
				stats.Errors = append(stats.Errors, err.Error())
				continue
			}

			err = mgmntClient.PushSchema(workspace, slug, schema, commit, commitMessage)
			if err != nil {
				if p != nil && cancel != nil {
					p.Stop("")
					cancel()
					p = nil
					cancel = nil
				}
				hasError = true
				log.WithError(err).Errorf("Failed to push schema %s", slug)
				stats.Failed++
				stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to push schema %s: %v", slug, err))
				continue
			}

			stats.Success++
			if p != nil && cancel != nil {
				p.Stop(fmt.Sprintf("Pushed schema: %s", slug))
				cancel()
				p = nil
				cancel = nil
			} else {
				fmt.Fprintf(os.Stdout, "Pushed schema: %s\n", slug)
			}
			hasError = false
		}

		fmt.Fprintf(os.Stdout, "\n=== Schema Push Summary ===\n")
		fmt.Fprintf(os.Stdout, "Total schemas processed: %d\n", stats.Total)
		fmt.Fprintf(os.Stdout, "Successfully pushed: %d\n", stats.Success)
		fmt.Fprintf(os.Stdout, "Failed to push: %d\n", stats.Failed)

		if stats.Failed > 0 {
			fmt.Fprintf(os.Stdout, "\nFailed schemas:\n")
			for _, errorMsg := range stats.Errors {
				fmt.Fprintf(os.Stdout, "  - %s\n", errorMsg)
			}
		}
		return nil
	},
}

func init() {
	schemaPushCmd.Flags().StringP("dir", "d", "", "Directory containing schema files (default: ./suprsend/schemas)")
	schemaPushCmd.Flags().BoolP("commit", "c", false, "Promote changes from draft to live after pushing")
	schemaPushCmd.Flags().String("commit-message", "", "Message describing the changes being committed")
	schemaPushCmd.PersistentFlags().StringP("slug", "g", "", "Schema slug to push (omit to push all)")
	schemaPushCmd.PersistentFlags().StringP("json", "j", "", `Schema definition as a JSON object (requires --slug). Must be a valid JSON Schema object, e.g. '{"type":"object","properties":{"key":{"type":"string"}}}'`)
	SchemaCmd.AddCommand(schemaPushCmd)
}
