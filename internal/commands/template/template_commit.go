/*
Copyright © 2025 SuprSend
*/
package template

import (
	"context"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/utils"
	"github.com/yarlson/pin"
)

var templateCommitCmd = &cobra.Command{
	Use:   "commit <slug>",
	Short: "Commit a template from draft to live",
	Long:  `Commit a template from draft to live in a workspace. Example: suprsend template commit <slug>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			log.Error("Template slug argument is required. Example: suprsend template commit <slug>")
			return fmt.Errorf("template slug argument is required. Example: suprsend template commit <slug>")
		}
		slug := args[0]

		workspace, _ := cmd.Flags().GetString("workspace")
		commitMessage, _ := cmd.Flags().GetString("commit-message")
		force, _ := cmd.Flags().GetBool("force")

		if commitMessage == "" {
			return fmt.Errorf("--commit-message (-m) is required")
		}

		mgmntClient := utils.GetSuprSendMgmntClient()

		var variants []map[string]any

		if force {
			var p *pin.Pin
			if !utils.IsOutputPiped() {
				p = pin.New(fmt.Sprintf("Validating template %s...", slug),
					pin.WithSpinnerColor(pin.ColorCyan),
					pin.WithTextColor(pin.ColorYellow),
				)
				cancel := p.Start(context.Background())
				defer cancel()
			}

			validateResp, err := mgmntClient.PreCommitValidate(workspace, slug)
			if err != nil {
				log.WithError(err).Errorf("Failed to pre-commit validate template %s", slug)
				return err
			}

			if p != nil {
				p.Stop("")
			}

			for _, v := range validateResp.Variants {
				if len(v.Errors) > 0 {
					log.Warnf("Skipping variant %s/%s for template %s due to errors: %v", v.Channel, v.ID, slug, v.Errors)
					continue
				}
				variants = append(variants, map[string]any{
					"channel": v.Channel,
					"id":      v.ID,
				})
			}

			if len(variants) == 0 {
				log.Errorf("No valid variants to commit for template %s", slug)
				return fmt.Errorf("no valid variants to commit for template %s", slug)
			}
		}

		var p *pin.Pin
		if !utils.IsOutputPiped() {
			p = pin.New(fmt.Sprintf("Committing template %s...", slug),
				pin.WithSpinnerColor(pin.ColorCyan),
				pin.WithTextColor(pin.ColorYellow),
			)
			cancel := p.Start(context.Background())
			defer cancel()
		}

		if err := mgmntClient.CommitTemplate(workspace, slug, commitMessage, variants); err != nil {
			log.WithError(err).Errorf("Failed to commit template %s", slug)
			return err
		}

		if p != nil {
			p.Stop(fmt.Sprintf("Successfully committed template '%s' to live", slug))
		} else {
			fmt.Fprintf(os.Stdout, "Successfully committed template '%s' to live\n", slug)
		}
		return nil
	},
}

func init() {
	templateCommitCmd.Flags().StringP("commit-message", "m", "", "Commit message describing the changes")
	templateCommitCmd.Flags().BoolP("force", "f", false, "Force commit by skipping variants with errors")
	TemplateCmd.AddCommand(templateCommitCmd)
}
