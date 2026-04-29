package commands

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

var genDocsCmd = &cobra.Command{
	Use:    "gendocs [dir]",
	Hidden: true,
	Short:  "Generate CLI documentation in Markdown",
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

		// Generate documentation
		err := doc.GenMarkdownTree(rootCmd, dir)

		// Restore the version command
		if versionCmd != nil {
			rootCmd.AddCommand(versionCmd)
		}

		return err
	},
}

func init() {
	genDocsCmd.SetHelpFunc(func(command *cobra.Command, strings []string) {
		command.Flags().MarkHidden("workspace")
		command.Flags().MarkHidden("service-token")
		command.Flags().MarkHidden("output")
		command.Flags().MarkHidden("verbosity")
		command.Flags().MarkHidden("no-color")
		command.Flags().MarkHidden("config")
		command.Parent().HelpFunc()(command, strings)
	})

	rootCmd.AddCommand(genDocsCmd)
}
