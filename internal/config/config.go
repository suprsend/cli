package config

import (
	"os"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Config holds the application's configuration.
type Config struct {
	CfgFile       string
	OutputType    string
	Verbosity     string
	ServiceToken  string
	NoColorOutput bool
	Workspace     string
	Quiet         bool
}

// cfg is the global configuration instance.
var Cfg = &Config{}

// initConfig reads in config file and ENV variables if set.
func InitConfig(cfgFile string) {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)

		if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
			log.Fatalf("Config file does not exist: %s", cfgFile)
		}
		if _, err := os.ReadFile(cfgFile); err != nil {
			log.Fatalf("Config file is not readable: %s - %v", cfgFile, err)
		}
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".suprsend")
	}

	viper.AutomaticEnv() // read in environment variables that match
	// load configs from env
	viper.BindEnv("debug", "DEBUG")
	// if NO_COLOR is set, disable color output
	if viper.GetBool("NO_COLOR") {
		color.NoColor = true
	}

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.Debug("Using config file:", viper.ConfigFileUsed())
	} else if cfgFile != "" {
		log.Fatalf("Failed to read config file: %s - %v", cfgFile, err)
	}
}

// setUpLogs set the log output ans the log level
func SetUpLogs() error {
	log.SetFormatter(&log.TextFormatter{
		DisableColors: viper.GetBool("NO_COLOR"),
		FullTimestamp: true,
		PadLevelText:  true,
	})
	if Cfg.OutputType == "json" {
		log.SetFormatter(&log.JSONFormatter{})
	}

	if Cfg.Quiet {
		log.SetOutput(os.Stderr)
		log.SetLevel(log.FatalLevel)
		return nil
	}
	if viper.GetBool("debug") {
		Cfg.Verbosity = "debug"
	}
	lvl, err := log.ParseLevel(Cfg.Verbosity)
	if err != nil {
		return errors.Wrap(err, "parsing log level")
	}
	log.SetOutput(os.Stderr)
	log.SetLevel(lvl)
	return nil
}
