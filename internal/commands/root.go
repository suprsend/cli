/*
Copyright © 2025 SuprSend
*/
package commands

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/commands/category"
	"github.com/suprsend/cli/internal/commands/event"
	"github.com/suprsend/cli/internal/commands/profiles"
	"github.com/suprsend/cli/internal/commands/schema"
	"github.com/suprsend/cli/internal/commands/template"
	"github.com/suprsend/cli/internal/commands/translation"
	workflow "github.com/suprsend/cli/internal/commands/workflow"
	"github.com/suprsend/cli/internal/commands/workspace"
	"github.com/suprsend/cli/internal/config"
	"github.com/suprsend/cli/internal/utils"
	"go.szostok.io/version/extension"
)

// rootCmd represents the base command when called without any subcommands.
//
// SilenceUsage suppresses cobra's "Usage: ..." dump when a subcommand's RunE
// returns an error. That dump is useful for argument-shape errors but pure
// noise on API / file / validation failures, which is what most of our
// errors are. cobra inherits this to every subcommand (see command.go:
// `if !cmd.SilenceUsage && !c.SilenceUsage` — root suppression wins).
var rootCmd = &cobra.Command{
	Use:          "suprsend",
	Short:        "CLI to interact with SuprSend, a Notification Infrastructure",
	SilenceUsage: true,
	Long: heredoc.Doc(`SuprSend is a robust notification infrastructure that helps you deploy multi-channel product notifications effortlessly and take care of user experience.

	This CLI lets you interact with your SuprSend workspace and do actions like fetching/modifying template, workflows etc.`),
}

// Execute runs the root command and handles structured error output.
func Execute() error {
	// Run early setup so flag-parse errors (unknown flags) also get proper
	// silencing and log formatting — PersistentPreRunE won't fire in that case.
	earlySetup()
	err := rootCmd.Execute()
	if err != nil {
		if isCobraUsageError(err) {
			err = clierr.Wrap(err, clierr.CodeInvalidUsage, "")
		}
		utils.WriteError(err)
	}
	return err
}

func isCobraUsageError(err error) bool {
	var notExist *pflag.NotExistError
	var valueRequired *pflag.ValueRequiredError
	var invalidValue *pflag.InvalidValueError
	var invalidSyntax *pflag.InvalidSyntaxError
	return errors.As(err, &notExist) ||
		errors.As(err, &valueRequired) ||
		errors.As(err, &invalidValue) ||
		errors.As(err, &invalidSyntax) ||
		strings.Contains(err.Error(), "unknown command")
}

// earlySetup scans raw os.Args to apply critical initialization before Cobra
// parses flags. This ensures correct behavior even when flag parsing fails.
func earlySetup() {
	conf := config.Cfg
	args := os.Args[1:]
	for i, arg := range args {
		switch {
		case arg == "--output=json" || arg == "-o=json":
			conf.OutputType = "json"
		case (arg == "--output" || arg == "-o") && i+1 < len(args) && args[i+1] == "json":
			conf.OutputType = "json"
		}
	}
	if config.ShouldJSONErrors() {
		rootCmd.SilenceErrors = true
		rootCmd.SilenceUsage = true
	}
}

func init() {
	var flags config.FlagValues

	rootCmd.Flags().StringVarP(&flags.Workspace, "workspace", "w", "staging", "Workspace name (e.g., staging, production)")
	rootCmd.PersistentFlags().StringVar(&flags.CfgFile, "config", "", "config file (default: $HOME/.suprsend.yaml)")
	rootCmd.PersistentFlags().StringVarP(&flags.OutputType, "output", "o", "pretty", "Output format: pretty, json, or yaml")
	rootCmd.PersistentFlags().StringVarP(&flags.Verbosity, "verbosity", "v", "info", "Log level (debug, info, warn, error, fatal, panic)")
	rootCmd.PersistentFlags().StringVarP(&flags.ServiceToken, "service-token", "s", "", "Service token (default: $SUPRSEND_SERVICE_TOKEN)")
	rootCmd.PersistentFlags().BoolVar(&flags.NoColor, "no-color", false, "Disable color output (default: $NO_COLOR)")
	rootCmd.PersistentFlags().BoolVarP(&flags.Quiet, "quiet", "q", false, "Suppress info/warn output (errors are still shown)")

	viper.BindPFlag("service_token", rootCmd.PersistentFlags().Lookup("service-token"))
	viper.BindPFlag("NO_COLOR", rootCmd.PersistentFlags().Lookup("no-color"))

	cobra.OnInitialize(func() {
		config.InitConfig(flags.CfgFile)
	})
	rootCmd.AddCommand(
		// 1. Register the 'version' command
		extension.NewVersionCobraCmd(
			// 2. Explicitly enable upgrade notice
			extension.WithUpgradeNotice("suprsend", "cli"),
		),
	)
	rootCmd.DisableAutoGenTag = true

	rootCmd.AddCommand(profiles.ProfileCmd)
	rootCmd.AddCommand(workflow.WorkflowCmd)
	rootCmd.AddCommand(category.CategoryCmd)
	rootCmd.AddCommand(event.EventCmd)
	rootCmd.AddCommand(translation.TranslationCmd)
	rootCmd.AddCommand(schema.SchemaCmd)
	rootCmd.AddCommand(template.TemplateCmd)
	rootCmd.AddCommand(workspace.WorkspaceCmd)

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		switch flags.OutputType {
		case "pretty", "json", "yaml":
		default:
			return clierr.New(
				fmt.Sprintf("invalid output format %q: must be pretty, json, or yaml", flags.OutputType),
				clierr.CodeInvalidUsage,
			)
		}

		if cmd.Name() == "gendocs" || cmd.Name() == "genskills" {
			return nil
		}
		if cmd.Name() == "version" || cmd.Name() == "help" || cmd.Name() == "env" {
			return nil
		}
		if cmd.Name() == "completion" || (cmd.Parent() != nil && cmd.Parent().Name() == "completion") {
			return nil
		}
		if cmd.Name() == "list-tools" && (cmd.Parent() != nil && cmd.Parent().Name() == "start-mcp-server") {
			return nil
		}
		if cmd.Name() == "profile" || (cmd.Parent() != nil && cmd.Parent().Name() == "profile") {
			return nil
		}

		if err := config.Cfg.Resolve(flags); err != nil {
			return err
		}

		if err := config.SetUpLogs(); err != nil {
			return err
		}

		utils.InitSDKWithUrls(
			config.Cfg.ServiceToken,
			config.Cfg.BaseUrl,
			config.Cfg.MgmntUrl,
			viper.GetBool("debug"),
		)

		return nil
	}
}
