package config

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/fatih/color"
	log "github.com/sirupsen/logrus"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/termio"
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
// Pass "" to fall back to Cfg.OutputType; pass a non-empty value to override
// when Cfg isn't populated yet (e.g. flag-parse-time error handling).
func ShouldJSONErrors(outputValue string) bool {
	if outputValue == "" {
		outputValue = Cfg.OutputType.Value
	}
	return outputValue == "json" || IsStderrPiped()
}

// hiddenWarnings collects warnings that were not shown on stderr because
// logging is suppressed (JSON errors mode). Reports print them on stdout or
// add them to the JSON result, so piped and -o json runs still see them.
type hiddenWarnings struct {
	mu   sync.Mutex
	msgs []string
}

func (h *hiddenWarnings) Levels() []log.Level { return []log.Level{log.WarnLevel} }

func (h *hiddenWarnings) Fire(e *log.Entry) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.msgs = append(h.msgs, e.Message)
	return nil
}

var warningSink = &hiddenWarnings{}

// HiddenWarnings returns, and clears, the warnings that were not shown on
// stderr. It is empty when logs go to stderr normally.
func HiddenWarnings() []string {
	warningSink.mu.Lock()
	defer warningSink.mu.Unlock()
	msgs := warningSink.msgs
	warningSink.msgs = nil
	return msgs
}

// SetUpLogs sets the log output and log level.
func SetUpLogs() error {
	log.SetFormatter(&cliFormatter{noColor: Cfg.NoColorOutput.Value})

	// In JSON errors mode, stderr carries only the structured error written by
	// utils.WriteError, so log lines are dropped. Warnings are kept for the
	// command's report. An explicit -v (or DEBUG) still logs to stderr.
	if (Cfg.OutputType.Value == "json" || IsStderrPiped()) && !Cfg.Verbosity.Set && !Cfg.Debug.Value {
		log.SetOutput(io.Discard)
		log.SetLevel(log.WarnLevel)
		log.AddHook(warningSink)
		return nil
	}

	if Cfg.Quiet.Value {
		log.SetOutput(termio.LogWriter(os.Stderr))
		log.SetLevel(log.ErrorLevel)
		return nil
	}
	if Cfg.Debug.Value {
		Cfg.Verbosity.Value = "debug"
	}
	lvl, err := log.ParseLevel(Cfg.Verbosity.Value)
	if err != nil {
		return clierr.Wrap(err, clierr.CodeInvalidUsage, fmt.Sprintf("invalid log level %q", Cfg.Verbosity.Value))
	}
	log.SetOutput(termio.LogWriter(os.Stderr))
	log.SetLevel(lvl)
	return nil
}
