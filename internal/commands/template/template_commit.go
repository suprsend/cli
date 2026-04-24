/*
Copyright © 2025 SuprSend
*/
package template

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/utils"
	"github.com/yarlson/pin"
)

var templateCommitCmd = &cobra.Command{
	Use:   "commit [<slug>]",
	Short: "Commit a template from draft to live",
	Long:  `Commit a template from draft to live in a workspace. Pass the template slug as a positional argument or via --slug. Example: suprsend template commit <slug>`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		slug := utils.ResolveSlug(cmd, args)
		if slug == "" {
			log.Error("template slug is required: provide it as a positional argument or via --slug")
			return fmt.Errorf("template slug is required: provide it as a positional argument or via --slug")
		}

		workspace, _ := cmd.Flags().GetString("workspace")
		commitMessage, _ := cmd.Flags().GetString("commit-message")
		force, _ := cmd.Flags().GetBool("force")

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		if dryRun {
			log.Infof("DRY RUN: would commit template '%s' to live", slug)
			return nil
		}

		if !force {
			msg := fmt.Sprintf("This will promote template '%s' to live in workspace \"%s\". Continue?", slug, workspace)
			confirmed, err := utils.ConfirmDestructiveAction(msg)
			if err != nil || !confirmed {
				log.Info("Aborted.")
				return nil
			}
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
			log.Infof("Successfully committed template '%s' to live", slug)
		}
		return nil
	},
}

func init() {
	templateCommitCmd.Flags().StringP("slug", "g", "", "Template slug")
	templateCommitCmd.Flags().String("commit-message", "", "Commit message describing the changes")
	templateCommitCmd.Flags().BoolP("force", "F", false, "Force commit by skipping variants with errors")
	templateCommitCmd.Flags().BoolP("dry-run", "n", false, "Print what would be committed without making any changes")
	TemplateCmd.AddCommand(templateCommitCmd)
}
