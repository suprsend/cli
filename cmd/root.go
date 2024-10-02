/*
Copyright © 2024 SuprSend

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	_ "fmt"
	"os"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.szostok.io/version/extension"
)

var (
	cfgFile    string
	outputType string
	verbosity  string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "suprsend",
	Short: "CLI to interact with SuprSend, a Notification Infrastructure",
	Long: `SuprSend is a robust notification infrastructure that helps you deploy multi-channel product notifications effortlessly and take care of user experience.

This CLI lets you interact with your SuprSend workspace and do actions like fetching/modifying template, workflows etc.`,
	Version: "0.0.1",
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.AddCommand(
		// 1. Register the 'version' command
		extension.NewVersionCobraCmd(
			// 2. Explicitly enable upgrade notice
			extension.WithUpgradeNotice("suprsend", "cli"),
		),
	)

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if err := SetUpLogs(verbosity, outputType); err != nil {
			return err
		}
		return nil
	}
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: $HOME/.suprsend.yaml)")
	rootCmd.PersistentFlags().StringVarP(&outputType, "output", "o", "pretty", "Output Tyle (pretty, yaml, json)")
	rootCmd.PersistentFlags().StringVarP(&verbosity, "verbosity", "v", "info", "Log level (debug, info, warn, error, fatal, panic)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".suprsend" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".suprsend")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.Debug("Using config file:", viper.ConfigFileUsed())
	}
}

// setUpLogs set the log output ans the log level
func SetUpLogs(level string, outputType string) error {
	if outputType == "json" {
		log.SetFormatter(&log.JSONFormatter{})
	}
	lvl, err := log.ParseLevel(verbosity)
	if err != nil {
		return errors.Wrap(err, "parsing log level")
	}
	log.SetLevel(lvl)
	return nil
}
