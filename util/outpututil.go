package util

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"text/tabwriter"

	// log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// OutputData chooses the output format based on the flag
func OutputData(data interface{}, format string) {
	switch format {
	case "json":
		outputJSON(data)
	case "yaml":
		outputYAML(data)
	default:
		outputTable(data)
	}
}

// outputJSON prints data in JSON format
func outputJSON(data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error creating JSON output:", err)
		return
	}
	fmt.Println(string(jsonData))
}

// outputYAML prints data in YAML format
func outputYAML(data interface{}) {
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		fmt.Println("Error creating YAML output:", err)
		return
	}
	fmt.Println(string(yamlData))
}

// outputText prints data in plain text format
// func outputText(data interface{}) {
// 	fmt.Printf("Data: %v\n", data)
// }

// FetchData simulates data retrieval
func FetchData() interface{} {
	return map[string]string{"key": "value"}
}

/*
outputTable prints a slice of structs or a single struct as a formatted table to the standard output.
It uses reflection to dynamically handle the struct fields and their values.

Parameters:
  - data: An interface{} that can be either a single struct or a slice of structs.

The function sets up a tab writer to format the output in a tabular form. It first checks if the input
is a slice; if not, it converts a single struct into a slice of one element. It then prints the struct
field names as the table headers and iterates over the slice to print each struct's field values as rows.

Note: The function assumes that the input is either a struct or a slice of structs. If the input is not
a struct or a slice of structs, it will print an error message and return.
*/
func outputTable(data interface{}) {
	// Set up the tab writer
	writer := tabwriter.NewWriter(os.Stdout, 10, 10, 2, ' ', 0)
	defer writer.Flush()

	// Reflect to handle both single struct and slice of structs
	v := reflect.ValueOf(data)
	if v.Kind() != reflect.Slice {
		// Convert single struct to slice of one element
		v = reflect.Append(reflect.MakeSlice(reflect.SliceOf(v.Type()), 0, 1), v)
	}

	// Handle headers (assumes structs, and that the first element is representative of all)
	if v.Len() > 0 {
		first := v.Index(0)
		if first.Kind() != reflect.Struct {
			fmt.Println("Error: Expected a struct")
			return
		}
		t := first.Type()
		// Print column headers
		for i := 0; i < first.NumField(); i++ {
			fmt.Fprintf(writer, "%s\t", t.Field(i).Name)
		}
		fmt.Fprintln(writer, "") // end headers row
	}

	// Print row data
	for i := 0; i < v.Len(); i++ {
		val := v.Index(i)
		for j := 0; j < val.NumField(); j++ {
			fmt.Fprintf(writer, "%v\t", val.Field(j).Interface())
		}
		fmt.Fprintln(writer, "") // end data row
	}
}
