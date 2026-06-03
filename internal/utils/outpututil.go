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
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"errors"

	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	log "github.com/sirupsen/logrus"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/config"
	"github.com/suprsend/cli/internal/termio"
	"github.com/tidwall/pretty"
	"github.com/yarlson/pin"
	"golang.org/x/term"
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
		// Wrapping pin's writer breaks its internal TTY detection (it checks for
		// *os.File). ShowSpinner has already verified stdout is a TTY, so it's
		// safe to force the interactive code path here.
		pin.ForceInteractive = true
		s.p = pin.New(text,
			pin.WithSpinnerColor(pin.ColorCyan),
			pin.WithTextColor(pin.ColorYellow),
			pin.WithWriter(termio.SpinnerWriter(os.Stdout)),
		)
		termio.MarkSpinnerStart()
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
		termio.MarkSpinnerStop()
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
	if slices.Contains(allowed, format) {
		return nil
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
	for line := range strings.SplitSeq(yamlString, "\n") {
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
	if val.Kind() == reflect.Pointer {
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

const (
	// defaultTerminalWidth is assumed when stdout is not a TTY (piped,
	// redirected, or under test) and $COLUMNS is unset.
	defaultTerminalWidth = 120
	// minTerminalWidth floors the detected width so a tiny/garbage value
	// can't collapse every column to nothing.
	minTerminalWidth = 40
	// maxColumnWidth caps any single column even when the terminal is very
	// wide, so long prose (e.g. a tool_description) wraps into readable
	// line lengths instead of one 250-char line.
	maxColumnWidth = 100
)

// terminalWidth reports the usable output width: the real terminal size when
// stdout is a TTY, else $COLUMNS, else defaultTerminalWidth. Floored at
// minTerminalWidth.
func terminalWidth() int {
	w := 0
	if cols, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && cols > 0 {
		w = cols
	} else if env := os.Getenv("COLUMNS"); env != "" {
		if parsed, err := strconv.Atoi(env); err == nil && parsed > 0 {
			w = parsed
		}
	}
	if w <= 0 {
		w = defaultTerminalWidth
	}
	if w < minTerminalWidth {
		w = minTerminalWidth
	}
	return w
}

// cellDisplayWidth is the width of the longest single line in a (possibly
// multi-line) cell, measured in runes so multibyte glyphs like "—" count as 1.
func cellDisplayWidth(s string) int {
	max := 0
	for line := range strings.SplitSeq(s, "\n") {
		if n := utf8.RuneCountInString(line); n > max {
			max = n
		}
	}
	return max
}

// allocateColumnWidths distributes a content-width budget across columns using
// max-min (water-filling) fairness: columns narrower than their fair share keep
// their natural width, and the freed space is redistributed to the wider
// columns. When everything fits in the budget, each column gets its natural
// width (no wrapping). Otherwise the widest columns absorb the squeeze.
func allocateColumnWidths(natural []int, budget int) []int {
	n := len(natural)
	w := make([]int, n)
	settled := make([]bool, n)
	remaining, left := budget, n
	for left > 0 {
		share := max(remaining/left, 1)
		progressed := false
		for i := range n {
			if !settled[i] && natural[i] <= share {
				w[i] = natural[i]
				remaining -= natural[i]
				settled[i] = true
				left--
				progressed = true
			}
		}
		if !progressed {
			// Every unsettled column wants more than its share: give each the
			// share, then hand any integer-division remainder to the widest.
			widest := -1
			for i := range n {
				if !settled[i] {
					w[i] = share
					remaining -= share
					if widest < 0 || natural[i] > natural[widest] {
						widest = i
					}
				}
			}
			if widest >= 0 && remaining > 0 {
				w[widest] += remaining
			}
			break
		}
	}
	return w
}

func printStructAsTable(values []reflect.Value) {
	if len(values) == 0 {
		log.Info("No data to display")
		return
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	// Borderless light style with column separators and a header line, to
	// approximate the previous tablewriter look. Left-align header + rows.
	style := table.StyleLight
	style.Options.DrawBorder = false
	style.Options.SeparateColumns = true
	style.Options.SeparateRows = false
	style.Options.SeparateHeader = true
	style.Options.SeparateFooter = false
	style.Format.Header = text.FormatDefault
	t.SetStyle(style)
	t.Style().Format.Header = text.FormatDefault

	elemType := values[0].Type()
	numFields := elemType.NumField()

	// Set headers based on struct field names, and seed each column's natural
	// width with its header width.
	headerRow := make(table.Row, 0, numFields)
	natural := make([]int, numFields)
	for i := range numFields {
		name := elemType.Field(i).Name
		headerRow = append(headerRow, name)
		natural[i] = cellDisplayWidth(name)
	}
	t.AppendHeader(headerRow)

	// Add rows, tracking each column's natural (unwrapped) width as we go.
	for _, val := range values {
		row := make(table.Row, 0, numFields)
		for i := range numFields {
			cell := formatValue(val.Field(i))
			row = append(row, cell)
			if w := cellDisplayWidth(cell); w > natural[i] {
				natural[i] = w
			}
		}
		t.AppendRow(row)
	}

	// Clamp natural widths to an absolute per-column ceiling, then fit the
	// total to the terminal. Overhead per column is 2 padding spaces plus a
	// separator; budget is the remaining width available to content.
	for i := range natural {
		if natural[i] > maxColumnWidth {
			natural[i] = maxColumnWidth
		}
	}
	budget := max(terminalWidth()-3*numFields,
		// degenerate terminal; let go-pretty wrap hard
		numFields)
	widths := allocateColumnWidths(natural, budget)

	colConfigs := make([]table.ColumnConfig, 0, numFields)
	for i := range numFields {
		colConfigs = append(colConfigs, table.ColumnConfig{
			Number:           i + 1,
			Align:            text.AlignLeft,
			AlignHeader:      text.AlignLeft,
			WidthMax:         widths[i],
			WidthMaxEnforcer: text.WrapSoft,
		})
	}
	t.SetColumnConfigs(colConfigs)

	t.Render()
}

func formatValue(v reflect.Value) string {
	switch v.Kind() {
	case reflect.Pointer:
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
