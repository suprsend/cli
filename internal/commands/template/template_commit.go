/*
Copyright © 2025 SuprSend
*/
package template

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/utils"
)

var templateCommitCmd = &cobra.Command{
	Use:   "commit [<slug>]",
	Short: "Commit a template from draft to live",
	Long:  `Commit a template from draft to live in a workspace. Pass the template slug as a positional argument or via --slug. Once committed, the template changes become visible to users.`,
	Example: `  # Commit a template to live (positional slug)
  suprsend template commit welcome-email

  # Commit using the flag form
  suprsend template commit --slug welcome-email

  # Dry run: see what would be committed without making changes
  suprsend template commit welcome-email --dry-run`,
	Annotations: map[string]string{
		"skills:tip.a-irreversible": "Commit is irreversible: it promotes the draft to **live**, overwriting the previous live version. Affected workflows immediately render the new content.",
		"skills:tip.b-inspect":      "If you didn't author the draft locally, run `suprsend template get --slug <slug> --mode draft` first to inspect what will become live.",
	},
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		slug := utils.ResolveSlug(cmd, args)
		if slug == "" {
			return clierr.New("template slug is required: provide it as a positional argument or via --slug", clierr.CodeInvalidUsage)
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
			validationSpinner := utils.NewSpinner(fmt.Sprintf("Validating template %s...", slug))

			validateResp, err := mgmntClient.PreCommitValidate(workspace, slug)
			if err != nil {
				log.WithError(err).Errorf("Failed to pre-commit validate template %s", slug)
				return clierr.Wrap(err, clierr.CodeAPIInternal, "")
			}

			validationSpinner.Stop("")

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
				return clierr.New(fmt.Sprintf("no valid variants to commit for template %s", slug), clierr.CodeAPIInternal)
			}
		}

		spinner := utils.NewSpinner(fmt.Sprintf("Committing template %s...", slug))

		if err := mgmntClient.CommitTemplate(workspace, slug, commitMessage, variants); err != nil {
			log.WithError(err).Errorf("Failed to commit template %s", slug)
			return clierr.Wrap(err, clierr.CodeAPIInternal, "")
		}

		spinner.Stop(fmt.Sprintf("Successfully committed template '%s' to live", slug))
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
