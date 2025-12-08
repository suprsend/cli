/*
Copyright © 2025 SuprSend
*/
package commands

import (
	"os"

	"github.com/MakeNowJust/heredoc/v2"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/suprsend/cli/internal/commands/category"
	"github.com/suprsend/cli/internal/commands/event"
	"github.com/suprsend/cli/internal/commands/profiles"
	"github.com/suprsend/cli/internal/commands/schema"
	"github.com/suprsend/cli/internal/commands/translation"
	workflow "github.com/suprsend/cli/internal/commands/workflow"
	"github.com/suprsend/cli/internal/config"
	"github.com/suprsend/cli/internal/utils"
	"go.szostok.io/version/extension"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "suprsend",
	Short: "CLI to interact with SuprSend, a Notification Infrastructure",
	Long: heredoc.Doc(`SuprSend is a robust notification infrastructure that helps you deploy multi-channel product notifications effortlessly and take care of user experience.

	This CLI lets you interact with your SuprSend workspace and do actions like fetching/modifying template, workflows etc.`),
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	conf := config.Cfg
	rootCmd.Flags().StringVarP(&conf.Workspace, "workspace", "w", "staging", "Workspace to use")
	rootCmd.PersistentFlags().StringVar(&conf.CfgFile, "config", "", "config file (default: $HOME/.suprsend.yaml)")
	rootCmd.Flags().StringVarP(&conf.OutputType, "output", "o", "pretty", "Output Style (pretty, yaml, json)")
	rootCmd.PersistentFlags().StringVarP(&conf.Verbosity, "verbosity", "v", "info", "Log level (debug, info, warn, error, fatal, panic)")
	rootCmd.Flags().StringVarP(&conf.ServiceToken, "service-token", "s", "", "Service token (default: $SUPRSEND_SERVICE_TOKEN)")
	rootCmd.PersistentFlags().BoolVarP(&conf.NoColorOutput, "no-color", "n", false, "Disable color output (default: $NO_COLOR)")

	viper.BindPFlag("service_token", rootCmd.PersistentFlags().Lookup("service-token"))
	viper.BindPFlag("NO_COLOR", rootCmd.PersistentFlags().Lookup("no-color"))
	//
	cobra.OnInitialize(func() {
		config.InitConfig(conf.CfgFile)
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

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if err := config.SetUpLogs(); err != nil {
			return err
		}
		// check the subcommand and return if it is gendocs
		if cmd.Name() == "gendocs" {
			return nil
		}

		if cmd.Name() == "profile" || (cmd.Parent() != nil && cmd.Parent().Name() == "profile") {
			return nil
		}

		// env > flag > config file -> profile
		serviceToken := getServiceTokenWithPriority()
		conf.ServiceToken = serviceToken

		utils.InitSDKWithUrls(
			conf.ServiceToken,
			profiles.GetResolvedBaseUrl(),
			profiles.GetResolvedMgmntUrl(),
			viper.GetBool("debug"),
		)
		return nil
	}
}

func getServiceTokenWithPriority() string {
	// ENV Variable
	if envToken := os.Getenv("SUPRSEND_SERVICE_TOKEN"); envToken != "" {
		log.Debug("Using service token from environment variable")
		return envToken
	}

	var cmdFlagToken string
	if viper.IsSet("service_token") {
		cmdFlagToken = viper.GetString("service_token")
	}

	if cmdFlagToken != "" {
		log.Debug("Using service token from command line flag")
		return cmdFlagToken
	}

	// Config file
	configPath := profiles.GetConfigFilePath()
	if configPath == "" {
		return ""
	}

	cfg, err := profiles.LoadConfig(configPath)
	if err != nil {
		return ""
	}

	activeProfile := cfg.Profiles[cfg.ActiveProfile]
	if activeProfile.ServiceToken != "" {
		log.Debug("Using service token from config file profile")
		return activeProfile.ServiceToken
	}

	// No token found
	log.Fatalln("No service token found in environment, command line, or config file")
	return ""
}
