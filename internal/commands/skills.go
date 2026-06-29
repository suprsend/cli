package commands

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

var genSkillsCmd = &cobra.Command{
	Use:    "genskills [dir]",
	Hidden: false,
	Short:  "Generate SKILLS.md",
	Args:   cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := args[0]

		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}

		// Temporarily remove the version command for documentation generation
		var versionCmd *cobra.Command
		for _, child := range rootCmd.Commands() {
			if child.Name() == "version" {
				versionCmd = child
				rootCmd.RemoveCommand(child)
				break
			}
		}

		// Generate skills
		doc.GenSkillsDir(rootCmd, dir, doc.SkillsConfig{
			Name:        "suprsend-cli",
			Description: "SuprSend CLI tool for managing SuprSend account and resources from the command line — workspaces, workflows, templates, categories, events, schemas, and translations. Use ONLY when the user wants the agent to RUN a `suprsend ...` command (e.g. \"push my template\", \"pull all workflows\", \"sync staging to prod\", \"generate Python types\") or asks about a specific CLI command's flags/behavior (\"what does `suprsend template commit --dry-run` do?\"). Do NOT load for documentation, lookup, or conceptual SuprSend questions (\"how does batching work\", \"what is a variant\", \"explain delivery nodes\") — load `suprsend-docs-support` for those. Do NOT load when the user is authoring workflow or template JSON without invoking the CLI — load `suprsend-workflow-schema` or `suprsend-template-schema` for those.",
			Metadata: map[string]string{
				"author":   "suprsend",
				"category": "cli",
			},
			Notes: []string{
				"Commands that return data (list, get) support `-o json` for machine-readable JSON output and `-o yaml` for YAML. Default `-o pretty` outputs a human-friendly table.",
				"The `profile` command and its subcommands (add, list, modify, remove, use) are only needed for self-hosted/BYOC SuprSend instances or managing multiple accounts. SaaS users do not need them. Profiles are not used for switching between workspaces within the same account; use the `--workspace` flag for that.",
			},
		})

		// Restore the version command
		if versionCmd != nil {
			rootCmd.AddCommand(versionCmd)
		}

		return nil
	},
}

func init() {
	genSkillsCmd.SetHelpFunc(func(command *cobra.Command, strings []string) {
		command.Flags().MarkHidden("workspace")
		command.Flags().MarkHidden("service-token")
		command.Flags().MarkHidden("output")
		command.Flags().MarkHidden("verbosity")
		command.Flags().MarkHidden("no-color")
		command.Flags().MarkHidden("config")
		command.Parent().HelpFunc()(command, strings)
	})

	rootCmd.AddCommand(genSkillsCmd)
}
