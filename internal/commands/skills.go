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
			Description: "SuprSend CLI is a command-line interface tool for managing your SuprSend account and resources. It provides a convenient way to interact with the SuprSend API, allowing you to perform various operations such as managing workspaces, users, workflow, templates and more.",
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
