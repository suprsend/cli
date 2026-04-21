package schema

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v5"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/suprsend/cli/mgmnt"
)

type SchemaWriteStats struct {
	Total   int
	Success int
	Failed  int
	Errors  []string
}

type SchemaPushStats struct {
	Total   int
	Success int
	Failed  int
	Errors  []string
}

type FilteredSchema struct {
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

func isDebugMode() bool {
	return viper.GetBool("debug")
}

func debugLog(format string, args ...interface{}) {
	if isDebugMode() {
		log.Infof(format, args...)
	}
}

func debugErrorLog(format string, args ...interface{}) {
	if isDebugMode() {
		log.Errorf(format, args...)
	}
}

func promptForOutputDirectory() string {
	reader := bufio.NewReader(os.Stdin)
	defaultDir := filepath.Join(".", "suprsend", "schemas")
	fmt.Fprintf(os.Stdout, "Where would you like to save the schema?\n")
	fmt.Fprintf(os.Stdout, "Default: %s\n", defaultDir)
	fmt.Fprintf(os.Stdout, "Enter directory path (or press Enter for default): ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultDir
	}
	return input
}

func ensureOutputDirectory(dirPath string) error {
	info, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stdout, "Creating directory: %s\n", dirPath)
			return os.MkdirAll(dirPath, 0o755)
		}
		return fmt.Errorf("error checking directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path '%s' exists but is not a directory", dirPath)
	}
	if info.Mode().Perm()&0o200 == 0 {
		return fmt.Errorf("directory '%s' is not writable", dirPath)
	}
	return nil
}

func validateInputDirectory(dirPath string) error {
	info, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("directory does not exist: %s", dirPath)
		}
		return fmt.Errorf("error checking directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path '%s' exists but is not a directory", dirPath)
	}
	if info.Mode().Perm()&0400 == 0 {
		return fmt.Errorf("directory '%s' is not readable", dirPath)
	}
	return nil
}

func filterSchemaData(schemas []mgmnt.SchemaResponse) []FilteredSchema {
	filtered := make([]FilteredSchema, len(schemas))
	for i, schema := range schemas {
		filtered[i] = FilteredSchema{
			Slug:        schema.Slug,
			Title:       schema.Title,
			Description: schema.Description,
		}
	}
	return filtered
}

func WriteSchemasToFiles(schemasResp *mgmnt.SchemasResponse, dirPath string) (*SchemaWriteStats, error) {
	stats := &SchemaWriteStats{
		Total:  len(schemasResp.Results),
		Errors: []string{},
	}

	info, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(dirPath, 0o755); err != nil {
				return stats, err
			}
		} else {
			return stats, fmt.Errorf("error accessing '%s': %v", dirPath, err)
		}
	} else if !info.IsDir() {
		return stats, err
	}

	for _, schema := range schemasResp.Results {
		obj, ok := schema.(map[string]any)
		if !ok {
			stats.Failed++
			stats.Errors = append(stats.Errors, "Invalid schema format")
			continue
		}

		slug, _ := obj["slug"].(string)
		slugDir := filepath.Join(dirPath, slug)
		if err := os.MkdirAll(slugDir, 0o755); err != nil {
			stats.Failed++
			stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to create directory for '%s': %v", slug, err))
			continue
		}

		if err := writeSchemaFiles(slugDir, slug, obj); err != nil {
			debugErrorLog("Error: %s", err)
			fmt.Fprintf(os.Stdout, "Error: Failed to write schema '%s': %v\n", slug, err)
			stats.Failed++
			stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to write schema '%s': %v", slug, err))
			continue
		}

		debugLog("Wrote: %s", slugDir)
		fmt.Fprintf(os.Stdout, "Wrote schema to %s\n", filepath.Join(slugDir, "schema.json"))
		stats.Success++
	}

	return stats, nil
}

// writeSchemaFiles splits a raw schema API response into schema.json (metadata)
// and payload_schema.json (extracted json_schema), writing both to slugDir.
func writeSchemaFiles(slugDir, slug string, obj map[string]any) error {
	schemaMeta := map[string]any{
		"$schema":     "https://schema.suprsend.com/schemas/v1/schema.json",
		"slug":        slug,
		"name":        obj["name"],
		"description": obj["description"],
	}
	metaData, err := json.MarshalIndent(schemaMeta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal schema.json: %w", err)
	}
	if err := os.WriteFile(filepath.Join(slugDir, "schema.json"), append(metaData, '\n'), 0o644); err != nil {
		return fmt.Errorf("write schema.json: %w", err)
	}

	jsonSchema, hasSchema := obj["json_schema"]
	if hasSchema && jsonSchema != nil {
		payloadMap, ok := jsonSchema.(map[string]any)
		if !ok {
			payloadMap = map[string]any{}
		}
		if _, hasRef := payloadMap["$schema"]; !hasRef {
			payloadMap["$schema"] = "https://json-schema.org/draft/2020-12/schema"
		}
		payloadData, err := json.MarshalIndent(payloadMap, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal payload_schema.json: %w", err)
		}
		if err := os.WriteFile(filepath.Join(slugDir, "payload_schema.json"), append(payloadData, '\n'), 0o644); err != nil {
			return fmt.Errorf("write payload_schema.json: %w", err)
		}
	}

	return nil
}

// ReadAndMergeSchemaFiles reads schema.json + payload_schema.json from slugDir,
// strips $schema keys, and returns the merged payload ready for the API.
func ReadAndMergeSchemaFiles(slugDir, slug string) (map[string]any, error) {
	schemaPath := filepath.Join(slugDir, "schema.json")
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("schemas/%s: failed to read schema.json: %w", slug, err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("schemas/%s: failed to parse schema.json: %w", slug, err)
	}
	delete(payload, "$schema")

	// path wins: ensure slug matches directory name
	if s, _ := payload["slug"].(string); s != slug {
		fmt.Fprintf(os.Stdout, "Warning: schemas/%s: schema.json#slug is %q but directory is %q — using directory value\n", slug, s, slug)
		payload["slug"] = slug
	}

	payloadSchemaPath := filepath.Join(slugDir, "payload_schema.json")
	if _, err := os.Stat(payloadSchemaPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("schemas/%s: missing payload_schema.json", slug)
	}
	psData, err := os.ReadFile(payloadSchemaPath)
	if err != nil {
		return nil, fmt.Errorf("schemas/%s: failed to read payload_schema.json: %w", slug, err)
	}
	var payloadSchema map[string]any
	if err := json.Unmarshal(psData, &payloadSchema); err != nil {
		return nil, fmt.Errorf("schemas/%s: failed to parse payload_schema.json: %w", slug, err)
	}
	delete(payloadSchema, "$schema")
	payload["json_schema"] = payloadSchema

	return payload, nil
}

type SchemasResponse struct {
	Schemas []SchemaResponse `json:"schemas"`
}

type SchemaResponse struct {
	Slug        string     `json:"slug"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	IsEnabled   bool       `json:"is_enabled"`
	JSONSchema  JSONSchema `json:"json_schema"`
}

type JSONSchema struct {
	Type       string              `json:"type"`
	Title      string              `json:"title"`
	Required   []string            `json:"required"`
	Properties map[string]Property `json:"properties"`
}

type Property struct {
	Type string `json:"type"`
}

// --- Public entrypoint you’ll call ---
func MergeAndValidate(baseSchemaStr string, patchJSON any) ([]byte, error) {
	var base map[string]any
	if err := json.Unmarshal([]byte(baseSchemaStr), &base); err != nil {
		return nil, fmt.Errorf("base schema unmarshal: %w", err)
	}

	patchMap, err := toMapAny(patchJSON)
	if err != nil {
		return nil, fmt.Errorf("patch decode: %w", err)
	}

	merged := MergeJSONSchemas(base, patchMap)
	merged = pruneNulls(merged).(map[string]any)
	ensureDraft(merged)

	mergedBytes, err := json.Marshal(merged)
	if err != nil {
		return nil, fmt.Errorf("marshal merged schema: %w", err)
	}

	if err := compileSchema(mergedBytes); err != nil {
		return nil, fmt.Errorf("merged schema invalid: %w", err)
	}

	return pretty(mergedBytes)
}

// Merge D into R under "properties.data" and validate the result.
// baseSchemaStr: R as JSON string
// dSchemaJSON:   D as map[string]any, string, or []byte
func MergeUnderDataAndValidate(baseSchemaStr string, dSchemaJSON any) ([]byte, error) {
	// Parse inputs
	var R map[string]any
	if err := json.Unmarshal([]byte(baseSchemaStr), &R); err != nil {
		return nil, fmt.Errorf("unmarshal R: %w", err)
	}
	D, err := toMapAny(dSchemaJSON)
	if err != nil {
		return nil, fmt.Errorf("decode D: %w", err)
	}

	// Hoist $defs/definitions from D to R root so `#/$defs/...` refs continue to resolve
	if dDefs := getMap(D["$defs"]); len(dDefs) > 0 {
		R["$defs"] = mergeStringKeyedSchemas(getMap(R["$defs"]), dDefs)
		delete(D, "$defs")
	}
	if dDefsOld := getMap(D["definitions"]); len(dDefsOld) > 0 { // legacy draft support
		// Normalize legacy `definitions` into root $defs
		R["$defs"] = mergeStringKeyedSchemas(getMap(R["$defs"]), dDefsOld)
		delete(D, "definitions")
	}

	// Ensure R.properties.data is an object schema we can merge into
	dataSchema := ensureDataObjectSchema(R) // returns the map at R.properties.data

	// Merge D into the existing data schema
	mergedData := MergeJSONSchemas(dataSchema, D)

	// Write back
	mustSetAt(R, []string{"properties", "data"}, mergedData)

	// Hygiene + validate
	R = pruneNulls(R).(map[string]any)
	normalizeObjectFields(R) // ensures $defs/properties/etc are {} not null
	ensureDraft(R)

	out, err := json.Marshal(R)
	if err != nil {
		return nil, err
	}
	if err := compileSchema(out); err != nil {
		return nil, fmt.Errorf("merged schema invalid: %w", err)
	}
	return pretty(out)
}

// ensureDataObjectSchema ensures R.properties.data exists and is an object schema map.
func ensureDataObjectSchema(R map[string]any) map[string]any {
	props := getMap(R["properties"])
	R["properties"] = props

	data := getMap(props["data"])
	if len(data) == 0 {
		data = map[string]any{}
		props["data"] = data
	}
	// Make it an object if not already typed
	if _, ok := data["type"]; !ok {
		data["type"] = "object"
	}
	// Ensure "properties" container exists for object schemas
	if _, ok := data["properties"]; !ok {
		data["properties"] = map[string]any{}
	}
	return data
}

// --- helpers specific to this "data" use-case ---
var objectFields = map[string]struct{}{
	"properties": {}, "patternProperties": {}, "$defs": {},
	"definitions": {}, "dependentSchemas": {}, "dependencies": {},
}

func normalizeObjectFields(m map[string]any) {
	for k, v := range m {
		switch vv := v.(type) {
		case map[string]any:
			normalizeObjectFields(vv)
		case []any:
			for i := range vv {
				if mm, ok := vv[i].(map[string]any); ok {
					normalizeObjectFields(mm)
				}
			}
		default:
		}
		// If an object-typed field is null/missing, coerce to empty object
		if _, isObj := objectFields[k]; isObj {
			if v == nil {
				m[k] = map[string]any{}
			}
		}
	}
}

// mustSetAt sets R[path...] = v, creating intermediate maps as needed.
func mustSetAt(m map[string]any, path []string, v any) {
	cur := m
	for i, k := range path {
		if i == len(path)-1 {
			cur[k] = v
			return
		}
		nxt := getMap(cur[k])
		cur[k] = nxt
		cur = nxt
	}
}

// --- Schema compile (this validates the schema itself) ---
func compileSchema(schemaBytes []byte) error {
	c := jsonschema.NewCompiler()
	// Make it strict and modern
	c.Draft = jsonschema.Draft2020
	c.ExtractAnnotations = true
	c.AssertFormat = true

	// Load from memory and compile; this will fail if the schema is malformed
	const resName = "mem://merged.schema.json"
	if err := c.AddResource(resName, bytes.NewReader(schemaBytes)); err != nil {
		return fmt.Errorf("add resource: %w", err)
	}
	_, err := c.Compile(resName)
	return err
}

// --- Merge logic (preserves addlProps:false, unions required, recursive) ---
func MergeJSONSchemas(base, add map[string]any) map[string]any {
	out := make(map[string]any, len(base)+len(add))
	for k, v := range base {
		out[k] = v
	}
	for k, vAdd := range add {
		switch k {
		case "properties", "patternProperties", "$defs":
			out[k] = mergeStringKeyedSchemas(getMap(out[k]), getMap(vAdd))
		case "required":
			out[k] = anySliceFromStrings(unionStrings(
				asStringSlice(out[k]),
				asStringSlice(vAdd),
			))
		case "allOf", "anyOf", "oneOf":
			out[k] = append(asAnySlice(out[k]), asAnySlice(vAdd)...)
		case "type":
			// Conservative: keep base "type" if present; otherwise take patch's.
			if out[k] == nil {
				out[k] = vAdd
			}
		default:
			// If both sides are maps, try recursive merge (for nested objects)
			if mb, ok := out[k].(map[string]any); ok {
				if ma, ok := vAdd.(map[string]any); ok {
					out[k] = MergeJSONSchemas(mb, ma)
					continue
				}
			}
			out[k] = vAdd
		}
	}
	return out
}

// --- Helpers ---
// call this right after MergeJSONSchemas and before compileSchema
func pruneNulls(v any) any {
	switch t := v.(type) {
	case map[string]any:
		for k, vv := range t {
			if vv == nil {
				delete(t, k)
				continue
			}
			t[k] = pruneNulls(vv)
		}
		return t
	case []any:
		out := make([]any, 0, len(t))
		for _, e := range t {
			if e == nil {
				continue
			}
			out = append(out, pruneNulls(e))
		}
		return out
	default:
		return v
	}
}

func mergeStringKeyedSchemas(a, b map[string]any) map[string]any {
	out := make(map[string]any, len(a)+len(b))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		if va, oka := v.(map[string]any); oka {
			if vb, okb := out[k].(map[string]any); okb {
				out[k] = MergeJSONSchemas(vb, va)
				continue
			}
		}
		out[k] = v
	}
	return out
}

func getMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func asAnySlice(v any) []any {
	if v == nil {
		return nil
	}
	if s, ok := v.([]any); ok {
		return s
	}
	return []any{v}
}

func asStringSlice(v any) []string {
	switch t := v.(type) {
	case nil:
		return nil
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, e := range t {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func unionStrings(a, b []string) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	out := make([]string, 0, len(a)+len(b))
	for _, x := range a {
		if _, ok := seen[x]; !ok {
			seen[x] = struct{}{}
			out = append(out, x)
		}
	}
	for _, x := range b {
		if _, ok := seen[x]; !ok {
			seen[x] = struct{}{}
			out = append(out, x)
		}
	}
	return out
}

func anySliceFromStrings(s []string) []any {
	res := make([]any, len(s))
	for i, v := range s {
		res[i] = v
	}
	return res
}

func ensureDraft(m map[string]any) {
	if _, ok := m["$schema"]; !ok {
		m["$schema"] = "https://json-schema.org/draft/2020-12/schema"
	}
}

func pretty(b []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := json.Indent(&buf, b, "", "  "); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func toMapAny(v any) (map[string]any, error) {
	switch t := v.(type) {
	case map[string]any:
		return t, nil
	case string:
		var m map[string]any
		if err := json.Unmarshal([]byte(t), &m); err != nil {
			return nil, err
		}
		return m, nil
	case []byte:
		var m map[string]any
		if err := json.Unmarshal(t, &m); err != nil {
			return nil, err
		}
		return m, nil
	default:
		return nil, errors.New("patch must be map[string]any, string, or []byte")
	}
}
