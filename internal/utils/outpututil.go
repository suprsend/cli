/*
Copyright © 2025 SuprSend
*/
package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/tidwall/pretty"
	"gopkg.in/yaml.v3"
)

// IsOutputPiped checks if os.Stdout is connected to a pipe or redirected.
func IsOutputPiped() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	// return true if --no-color is set to be true
	if viper.GetBool("NO_COLOR") {
		return true
	}

	// If ModeCharDevice is NOT set, it means the output is not a character device (terminal).
	// This implies it's a pipe or redirection.
	return (fi.Mode() & os.ModeCharDevice) == 0
}

func supportsColor() bool {
	// check if output is redirected to a file
	fileInfo, _ := os.Stdout.Stat()
	if (fileInfo.Mode() & os.ModeCharDevice) == 0 {
		return false
	}

	if viper.GetBool("NO_COLOR") {
		return false
	}

	return true
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
			fmt.Println("No data to display")
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
		fmt.Println("No data to display")
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
