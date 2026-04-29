/*
Copyright © 2025 SuprSend
*/
package main

import (
	_ "embed"
	"errors"
	"os"

	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		var ce *clierr.CLIError
		if errors.As(err, &ce) {
			os.Exit(ce.ExitCode())
		}
		os.Exit(clierr.ExitGeneralError)
	}
}
