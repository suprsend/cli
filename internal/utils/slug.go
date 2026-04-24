package utils

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func ResolveSlug(cmd *cobra.Command, args []string) string {
	flagVal, _ := cmd.Flags().GetString("slug")
	if len(args) > 0 {
		if flagVal != "" && flagVal != args[0] {
			fmt.Fprintf(os.Stderr,
				"warning: slug provided as both positional (%q) and --slug (%q); using positional\n",
				args[0], flagVal)
		}
		return args[0]
	}
	return flagVal
}
