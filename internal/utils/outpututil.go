/*
Copyright © 2025 SuprSend
*/
package utils

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"errors"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
	log "github.com/sirupsen/logrus"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/config"
	"github.com/tidwall/pretty"
	"github.com/yarlson/pin"
	"gopkg.in/yaml.v3"
)

// ConfirmDestructiveAction prints prompt + "[y/N]" on stderr and reads the answer from stdin.
// Returns true if the user confirmed. Skips the prompt and returns true when stdin is not a TTY.
func ConfirmDestructiveAction(prompt string) (bool, error) {
	fi, err := os.Stdin.Stat()
	if err != nil || (fi.Mode()&os.ModeCharDevice) == 0 {
		return true, nil
	}
	fmt.Fprintf(os.Stderr, "%s [y/N] ", prompt)
	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes", nil
}

// IsInputInteractive returns true when stdin is a TTY — i.e. a human can respond to prompts.
// Use this to gate any interactive prompt; IsOutputPiped is for color/spinner decisions only.
func IsInputInteractive() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// IsOutputPiped checks if os.Stdout is connected to a pipe or redirected.
func IsOutputPiped() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	// return true if --no-color is set to be true
	if config.Cfg.NoColorOutput.Value {
		return true
	}

	// If ModeCharDevice is NOT set, it means the output is not a character device (terminal).
	// This implies it's a pipe or redirection.
	return (fi.Mode() & os.ModeCharDevice) == 0
}

// ShowSpinner returns true when a spinner should be displayed — i.e. output is not piped, quiet mode is off, and output format is not JSON.
func ShowSpinner() bool {
	return !IsOutputPiped() && !config.Cfg.Quiet.Value && config.Cfg.OutputType.Value != "json"
}

// Spinner is a thin wrapper around pin.Pin that is nil-safe and no-ops when quiet/piped.
type Spinner struct {
	p      *pin.Pin
	cancel context.CancelFunc
}

// NewSpinner creates and starts a spinner with the given text. Returns a no-op Spinner when output is piped or quiet mode is on.
func NewSpinner(text string) *Spinner {
	s := &Spinner{}
	if ShowSpinner() {
		s.p = pin.New(text,
			pin.WithSpinnerColor(pin.ColorCyan),
			pin.WithTextColor(pin.ColorYellow),
		)
		s.cancel = s.p.Start(context.Background())
	}
	return s
}

// Stop stops the spinner with the given message. Safe to call multiple times or on a no-op Spinner.
func (s *Spinner) Stop(msg string) {
	if s.p != nil {
		s.p.Stop(msg)
		s.cancel()
		s.p = nil
	}
}

// UpdateMessage replaces the spinner's text in place. No-op when the spinner
// is suppressed (quiet, piped output).
func (s *Spinner) UpdateMessage(msg string) {
	if s.p != nil {
		s.p.UpdateMessage(msg)
	}
}

// WriteError writes err to stderr as a structured JSON CLIError when in JSON errors mode.
// Pass outputType when the call site runs before Resolve has populated Cfg.OutputType
// (e.g. flag-parse-time errors); pass "" otherwise.
func WriteError(err error, outputType string) {
	if err == nil || !config.ShouldJSONErrors(outputType) {
		return
	}
	var ce *clierr.CLIError
	if !errors.As(err, &ce) {
		ce = clierr.Wrap(err, clierr.CodeUnknown, "")
	}
	fmt.Fprintln(os.Stderr, string(ce.JSON()))
}

func supportsColor() bool {
	// check if output is redirected to a file
	fileInfo, _ := os.Stdout.Stat()
	if (fileInfo.Mode() & os.ModeCharDevice) == 0 {
		return false
	}

	if config.Cfg.NoColorOutput.Value {
		return false
	}

	return true
}

// ValidateOutputType returns a clierr if format is not one of the allowed values.
func ValidateOutputType(format string, allowed ...string) error {
	for _, a := range allowed {
		if format == a {
			return nil
		}
	}
	return clierr.New(
		fmt.Sprintf("invalid output format %q: must be one of %v", format, allowed),
		clierr.CodeInvalidUsage,
	)
}

// OutputData chooses the output format based on the flag
func OutputData(data any, format string) {
	switch format {
	case "json":
		outputJSON(data)
	case "yaml":
		outputYAML(data)
	default:
		// outputJSON(data)
		outputTable(data)
	}
}

// outputJSON prints data in JSON format
func outputJSON(data any) {
	jsonData, err := json.MarshalIndent(data, "", "   ")
	if err != nil {
		log.Fatal("Error creating JSON output:", err)
		return
	}

	if supportsColor() || !IsOutputPiped() {
		fmt.Println(string(pretty.Color(jsonData, nil)))
	} else {
		fmt.Println(string(jsonData))
	}
}

func colorizeYAML(yamlString string) string {
	lines := []string{}
	for _, line := range strings.Split(yamlString, "\n") {
		if idx := strings.Index(line, ":"); idx != -1 {
			// Preserve leading spaces (indentation)
			leading := line[:idx]
			keyAndRest := line[idx:]
			// Find the key (after leading spaces, before colon)
			key := leading + color.HiBlueString(strings.TrimSpace(line[len(leading):idx]))
			// Colorize value if present
			value := ""
			if len(keyAndRest) > 1 {
				value = color.GreenString(keyAndRest[1:])
			}
			lines = append(lines, key+":"+value)
		} else {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

// outputYAML prints data in YAML format
func outputYAML(data any) {
	var node yaml.Node
	err := node.Encode(data)
	if err != nil {
		log.Fatal("Error encoding YAML node:", err)
		return
	}

	var b strings.Builder
	encoder := yaml.NewEncoder(&b)
	encoder.SetIndent(4)
	err = encoder.Encode(&node)
	encoder.Close()
	if err != nil {
		log.Fatal("Error creating YAML output:", err)
		return
	}
	// trim the yaml data
	yamlString := strings.TrimSpace(b.String())
	if supportsColor() {
		fmt.Println(colorizeYAML(yamlString))
	} else {
		fmt.Println(yamlString)
	}
}

func outputTable(data any) {
	val := reflect.ValueOf(data)

	// If the input is a pointer, get the underlying element
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// Handle single struct
	if val.Kind() == reflect.Struct {
		printStructAsTable([]reflect.Value{val})
		return
	}

	// Handle slice of structs
	if val.Kind() == reflect.Slice {
		if val.Len() == 0 {
			log.Info("No data to display")
			return
		}
		elemType := val.Index(0).Type()
		if elemType.Kind() != reflect.Struct {
			log.Fatal("Slice elements must be structs")
			return
		}
		values := make([]reflect.Value, val.Len())
		for i := 0; i < val.Len(); i++ {
			values[i] = val.Index(i)
		}
		printStructAsTable(values)
		return
	}

	log.Fatal("Input must be a struct or a slice of structs")
}

func printStructAsTable(values []reflect.Value) {
	if len(values) == 0 {
		log.Info("No data to display")
		return
	}

	table := tablewriter.NewTable(os.Stdout,
		tablewriter.WithRenderer(renderer.NewBlueprint(tw.Rendition{
			Borders: tw.Border{
				Left:   tw.Off,
				Right:  tw.Off,
				Top:    tw.Off,
				Bottom: tw.Off,
			},
			Settings: tw.Settings{
				Separators: tw.Separators{BetweenRows: tw.Off, BetweenColumns: tw.On, ShowHeader: tw.Off, ShowFooter: tw.Off},
				Lines: tw.Lines{
					ShowTop:        tw.Off,
					ShowBottom:     tw.Off,
					ShowHeaderLine: tw.On,
					ShowFooterLine: tw.Off,
				},
			},
		})),
		tablewriter.WithConfig(tablewriter.Config{
			Header: tw.CellConfig{
				Formatting: tw.CellFormatting{Alignment: tw.AlignLeft},
			},
			Row: tw.CellConfig{
				Formatting: tw.CellFormatting{
					MergeMode: tw.MergeNone,
					Alignment: tw.AlignLeft,
				},
			},
		}),
	)

	// Set headers based on struct field names
	elemType := values[0].Type()
	var headers []string
	for i := 0; i < elemType.NumField(); i++ {
		headers = append(headers, elemType.Field(i).Name)
	}
	table.Header(headers)

	// Add rows
	var rows [][]any
	for _, val := range values {
		var row []any
		for i := 0; i < val.NumField(); i++ {
			field := val.Field(i)
			row = append(row, formatValue(field))
		}
		rows = append(rows, row)
	}
	table.Bulk(rows)

	table.Render()
}

func formatValue(v reflect.Value) string {
	switch v.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			return "null"
		}
		return formatValue(v.Elem())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', 2, 64)
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	case reflect.Map:
		if v.IsNil() {
			return ""
		}
		// Check if it's map[string]any
		if v.Type().Key().Kind() == reflect.String && v.Type().Elem().Kind() == reflect.Interface {
			m := v.Interface()
			if b, err := json.Marshal(m); err == nil {
				return string(b)
			}
		}
		return fmt.Sprintf("%v", v.Interface())
	default:
		if !v.IsValid() || (v.Kind() == reflect.Interface && v.IsNil()) {
			return ""
		}
		return fmt.Sprintf("%v", v.Interface())
	}
}
