package config

import (
	"bytes"
	"fmt"
	"os"
	"sync"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type cliFormatter struct {
	noColor bool
}

func (f *cliFormatter) Format(entry *log.Entry) ([]byte, error) {
	var buf bytes.Buffer
	if entry.Level == log.InfoLevel {
		fmt.Fprintf(&buf, "%s\n", entry.Message)
		return buf.Bytes(), nil
	}
	var levelLabel string
	if f.noColor {
		levelLabel = fmt.Sprintf("%-5s", entry.Level.String())
	} else {
		switch entry.Level {
		case log.WarnLevel:
			levelLabel = color.YellowString("%-5s", entry.Level.String())
		case log.ErrorLevel:
			levelLabel = color.RedString("%-5s", entry.Level.String())
		case log.DebugLevel:
			levelLabel = color.CyanString("%-5s", entry.Level.String())
		default:
			levelLabel = fmt.Sprintf("%-5s", entry.Level.String())
		}
	}
	fmt.Fprintf(&buf, "%s %s\n", levelLabel, entry.Message)
	return buf.Bytes(), nil
}

// Config holds the application's configuration.
type Config struct {
	CfgFile       string
	OutputType    string
	Verbosity     string
	ServiceToken  ConfigString
	NoColorOutput bool
	Workspace     string
	Quiet         bool
	BaseUrl       ConfigString
	MgmntUrl      ConfigString
	ProxyURL      ConfigString
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

var isStderrPiped = sync.OnceValue(func() bool {
	fi, err := os.Stderr.Stat()
	return err == nil && (fi.Mode()&os.ModeCharDevice) == 0
})

// IsStderrPiped reports whether os.Stderr is not connected to a terminal.
// Result is cached after the first call.
func IsStderrPiped() bool {
	return isStderrPiped()
}

// ShouldJSONErrors returns true when errors must be emitted as structured JSON.
func ShouldJSONErrors() bool {
	return Cfg.OutputType == "json" || IsStderrPiped()
}

// setUpLogs set the log output ans the log level
func SetUpLogs() error {
	log.SetFormatter(&cliFormatter{noColor: viper.GetBool("NO_COLOR")})

	// In JSON errors mode suppress logrus entirely — utils.WriteError is the sole stderr writer.
	if Cfg.OutputType == "json" || IsStderrPiped() {
		log.SetLevel(log.FatalLevel)
		return nil
	}

	if Cfg.Quiet {
		log.SetOutput(os.Stderr)
		log.SetLevel(log.ErrorLevel)
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
