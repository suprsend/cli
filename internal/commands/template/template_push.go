/*
Copyright © 2025 SuprSend
*/
package template

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/utils"
	"github.com/yarlson/pin"
)

var templatePushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push templates and their variants from local to SuprSend workspace",
	Long:  `Push templates and their variants from local to SuprSend workspace`,
	Run: func(cmd *cobra.Command, args []string) {
		workspace, _ := cmd.Flags().GetString("workspace")
		path, _ := cmd.Flags().GetString("dir")
		commit, _ := cmd.Flags().GetString("commit")
		commitMessage, _ := cmd.Flags().GetString("commit-message")
		slug, _ := cmd.Flags().GetString("slug")

		if path == "" {
			path = filepath.Join(".", "suprsend", "templates")
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			log.Errorf("Directory %s does not exist", path)
			return
		}

		mgmntClient := utils.GetSuprSendMgmntClient()

		stats := &TemplatePushStats{
			Errors: []string{},
		}

		hasError := false
		var p *pin.Pin
		var cancel context.CancelFunc

		if slug != "" {
			// Push a single template by slug
			stats.Total = 1
			templateDir := filepath.Join(path, slug)

			if _, err := os.Stat(templateDir); os.IsNotExist(err) {
				log.Errorf("Template directory %s does not exist", templateDir)
				stats.Failed++
				stats.Errors = append(stats.Errors, fmt.Sprintf("Template directory %s does not exist", templateDir))
			} else {
				if !utils.IsOutputPiped() {
					p = pin.New(fmt.Sprintf("Pushing template %s...", slug),
						pin.WithSpinnerColor(pin.ColorCyan),
						pin.WithTextColor(pin.ColorYellow),
					)
					cancel = p.Start(context.Background())
				}

				variants, err := readTemplateVariants(templateDir)
				if err != nil {
					log.WithError(err).Errorf("Failed to read template %s", slug)
					stats.Failed++
					stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to read template %s: %v", slug, err))
				} else {
					var pushFailed bool
					for _, variant := range variants {
						if err := mgmntClient.PushTemplateVariant(workspace, slug, variant, commit, commitMessage); err != nil {
							log.WithError(err).Errorf("Failed to push variant for template %s", slug)
							stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to push variant for template %s: %v", slug, err))
							pushFailed = true
						}
					}
					if pushFailed {
						stats.Failed++
					} else {
						stats.Success++
					}
				}

				if p != nil && cancel != nil {
					if stats.Success > 0 {
						p.Stop(fmt.Sprintf("Pushed template: %s (%d variants)", slug, len(variants)))
					} else {
						p.Stop("")
					}
					cancel()
				} else if stats.Success > 0 {
					fmt.Fprintf(os.Stdout, "Pushed template: %s\n", slug)
				}
			}
		} else {
			// Push all templates
			entries, err := os.ReadDir(path)
			if err != nil {
				log.WithError(err).Errorf("Failed to read templates directory")
				return
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

				templateSlug := entry.Name()
				templateDir := filepath.Join(path, templateSlug)

				if !hasError && !utils.IsOutputPiped() {
					p = pin.New(fmt.Sprintf("Pushing template %s...", templateSlug),
						pin.WithSpinnerColor(pin.ColorCyan),
						pin.WithTextColor(pin.ColorYellow),
					)
					cancel = p.Start(context.Background())
				}

				variants, err := readTemplateVariants(templateDir)
				if err != nil {
					if p != nil && cancel != nil {
						p.Stop("")
						cancel()
						p = nil
						cancel = nil
					}
					hasError = true
					log.WithError(err).Errorf("Failed to read template %s", templateSlug)
					stats.Failed++
					stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to read template %s: %v", templateSlug, err))
					continue
				}

				var pushFailed bool
				for _, variant := range variants {

					if err := mgmntClient.PushTemplateVariant(workspace, templateSlug, variant, commit, commitMessage); err != nil {

						log.WithError(err).Errorf("Failed to push variant for template %s", templateSlug)
						stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to push variant for template %s: %v", templateSlug, err))
						pushFailed = true
					}
				}

				if pushFailed {
					if p != nil && cancel != nil {
						p.Stop("")
						cancel()
						p = nil
						cancel = nil
					}
					hasError = true
					stats.Failed++
					continue
				}

				stats.Success++
				if p != nil && cancel != nil {
					p.Stop(fmt.Sprintf("Pushed template: %s (%d variants)", templateSlug, len(variants)))
					cancel()
					p = nil
					cancel = nil
				} else {
					fmt.Fprintf(os.Stdout, "Pushed template: %s\n", templateSlug)
				}
				hasError = false
			}
		}

		fmt.Fprintf(os.Stdout, "\n=== Template Push Summary ===\n")
		fmt.Fprintf(os.Stdout, "Total templates processed: %d\n", stats.Total)
		fmt.Fprintf(os.Stdout, "Successfully pushed: %d\n", stats.Success)
		fmt.Fprintf(os.Stdout, "Failed to push: %d\n", stats.Failed)

		if stats.Failed > 0 {
			fmt.Fprintf(os.Stdout, "\nFailed templates:\n")
			for _, errorMsg := range stats.Errors {
				fmt.Fprintf(os.Stdout, "  - %s\n", errorMsg)
			}
		}
	},
}

// readTemplateVariants walks a template directory and returns all reassembled variants.
func readTemplateVariants(templateDir string) ([]map[string]any, error) {
	var variants []map[string]any

	err := filepath.WalkDir(templateDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "variant.json" {
			return nil
		}

		variantDir := filepath.Dir(path)
		variant, err := readAndAssembleVariant(variantDir)
		if err != nil {
			return fmt.Errorf("failed to read variant at %s: %w", variantDir, err)
		}

		variants = append(variants, variant)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return variants, nil
}

func init() {
	templatePushCmd.PersistentFlags().StringP("dir", "d", "", "Input directory for templates (default: ./suprsend/templates)")
	templatePushCmd.PersistentFlags().StringP("commit", "c", "false", "Commit the templates (--commit=true)")
	templatePushCmd.PersistentFlags().StringP("commit-message", "m", "", "Commit message describing the changes")
	templatePushCmd.PersistentFlags().StringP("slug", "g", "", "Slug of a specific template to push")
	TemplateCmd.AddCommand(templatePushCmd)
}
