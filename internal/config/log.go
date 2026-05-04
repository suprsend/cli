package config

import (
	"bytes"
	"fmt"
	"os"
	"sync"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
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
func ShouldJSONErrors(outputValue ...string) bool {
	v := Cfg.OutputType.Value
	if len(outputValue) > 0 {
		v = outputValue[0]
	}
	return v == "json" || IsStderrPiped()
}

// SetUpLogs sets the log output and log level.
func SetUpLogs() error {
	log.SetFormatter(&cliFormatter{noColor: Cfg.NoColorOutput.Value})

	// In JSON errors mode suppress logrus entirely — utils.WriteError is the sole stderr writer.
	if Cfg.OutputType.Value == "json" || IsStderrPiped() {
		log.SetLevel(log.FatalLevel)
		return nil
	}

	if Cfg.Quiet.Value {
		log.SetOutput(os.Stderr)
		log.SetLevel(log.ErrorLevel)
		return nil
	}
	if Cfg.Debug.Value {
		Cfg.Verbosity.Value = "debug"
	}
	lvl, err := log.ParseLevel(Cfg.Verbosity.Value)
	if err != nil {
		return errors.Wrap(err, "parsing log level")
	}
	log.SetOutput(os.Stderr)
	log.SetLevel(lvl)
	return nil
}
