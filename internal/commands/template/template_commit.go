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

  # Commit only the variants that pass validation
  suprsend template commit welcome-email --skip-invalid

  # Dry run: see what would be committed without making changes
  suprsend template commit welcome-email --dry-run`,
	Annotations: map[string]string{
		"skills:tip.a-irreversible": "Commit is irreversible: it promotes the draft to **live**, overwriting the previous live version. Affected workflows immediately render the new content.",
		"skills:tip.b-inspect":      "If you didn't author the draft locally, run `suprsend template get --slug <slug> --mode draft` first to inspect what will become live.",
		"skills:tip.c-invalid":      "All changed variants are committed together, and one invalid variant fails the commit. `--skip-invalid` runs a pre-commit validation and commits only the variants that pass; the skipped ones are listed in the result (the command fails if every changed variant fails).",
		"skills:tip.d-confirm":      "The confirmation prompt only appears when stdin is a TTY. In non-interactive shells (CI, agents) the commit proceeds without asking; `-F/--force` skips it on a TTY too. Use `--dry-run` if you only want to check the command.",
		"skills:tip.e-result":       "The result says `committed`, or `unchanged` when the draft already matched live. With `-o json` it is one JSON object with `items` and `counts`.",
	},
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		slug := utils.ResolveSlug(cmd, args)
		if slug == "" {
			return clierr.New("template slug is required: provide it as a positional argument or via --slug", clierr.CodeInvalidUsage)
		}

		workspace, _ := cmd.Flags().GetString("workspace")
		commitMessage, _ := cmd.Flags().GetString("commit-message")
		force, _ := cmd.Flags().GetBool("force")
		skipInvalid, _ := cmd.Flags().GetBool("skip-invalid")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		report := utils.NewReport("template commit", workspace, dryRun)
		if dryRun {
			report.Add("templates/"+slug, utils.StatusWouldCommit, "")
			return report.Finish(nil)
		}

		if !force {
			msg := fmt.Sprintf("This will promote template '%s' to live in workspace \"%s\". Continue?", slug, workspace)
			confirmed, err := utils.ConfirmDestructiveAction(msg)
			if err != nil || !confirmed {
				log.Info("Aborted.")
				return nil
			}
		}

		spinner := utils.NewSpinner(fmt.Sprintf("Committing template %s...", slug))
		status, detail, err := commitTemplate(utils.GetSuprSendMgmntClient(), workspace, slug, commitMessage, skipInvalid)
		spinner.Stop("")
		if err != nil {
			report.Fail("templates/"+slug, err)
			return report.Finish(err)
		}
		report.Add("templates/"+slug, status, detail)
		return report.Finish(nil)
	},
}

func init() {
	templateCommitCmd.Flags().StringP("slug", "g", "", "Template slug")
	templateCommitCmd.Flags().String("commit-message", "", "Commit message describing the changes")
	templateCommitCmd.Flags().BoolP("force", "F", false, "Skip the confirmation prompt")
	templateCommitCmd.Flags().Bool("skip-invalid", false, "Commit only the variants that pass pre-commit validation and list the ones skipped")
	templateCommitCmd.Flags().BoolP("dry-run", "n", false, "Report what would be committed without making any changes")
	TemplateCmd.AddCommand(templateCommitCmd)
}
