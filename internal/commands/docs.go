package commands

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// generateDocs writes one Markdown page per visible command to dir.
// GenMarkdownTree only creates and overwrites files, so previously generated
// pages (suprsend.md, suprsend_*.md) are removed first; otherwise pages for
// removed or renamed commands would linger. Other files in dir are kept.
func generateDocs(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, pattern := range []string{rootCmd.Name() + ".md", rootCmd.Name() + "_*.md"} {
		stale, err := filepath.Glob(filepath.Join(dir, pattern))
		if err != nil {
			return err
		}
		for _, f := range stale {
			if err := os.Remove(f); err != nil {
				return err
			}
		}
	}
	return doc.GenMarkdownTree(rootCmd, dir)
}

var genDocsCmd = &cobra.Command{
	Use:    "gendocs [dir]",
	Hidden: true,
	Short:  "Generate CLI documentation in Markdown",
	Args:   cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return generateDocs(args[0])
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
