package commands

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/cli/mgmnt"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type schemaToBeGeneratedType struct {
	Slug        string           `json:"slug"`
	VersionNo   *int             `json:"version_no"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	JSONSchema  mgmnt.JSONSchema `json:"json_schema"`
	TriggerName string           `json:"trigger_name"`
}

var generateTypesCmd = &cobra.Command{
	Use:   "generate-types",
	Short: "Generate type definitions from JSON Schema",
	Long:  "Generate typed code from trigger payload JSON schemas. Fetches schemas linked to workflows and events from a workspace and generates type definitions in the target language.",
	Annotations: map[string]string{
		"skills:tip.a-after-commit": "Run after every `suprsend schema commit` so consuming code stays in sync with the live trigger payloads.",
		"skills:tip.b-language":     "Pick the right subcommand for your stack: `python` (Pydantic), `typescript` (optional Zod), `go`, `java`, `kotlin`, `swift`, or `dart`.",
	},
}

var generateTypesPythonCmd = &cobra.Command{
	Use:   "python [flags]",
	Short: "Generate Python types from JSON Schema",
	Long:  "Generate Python type definitions from trigger payload schemas. Fetches linked schemas from the workspace and generates a single Python file with type classes. Supports Pydantic models (enabled by default).",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		pydantic, _ := cmd.Flags().GetBool("pydantic")
		if pydantic {
			cmd.Flags().Set("build-flags", "just-types=true,python-version=3.7,pydantic-base-model=true")
		} else {
			cmd.Flags().Set("build-flags", "just-types=true,python-version=3.7")
		}
		generateTypesForLanguage("python")(cmd, args)
	},
}

var generateTypesJavaCmd = &cobra.Command{
	Use:   "java [flags]",
	Short: "Generate Java types from JSON Schema",
	Long:  "Generate Java type definitions from trigger payload schemas. Creates one Java file per linked schema in a package directory structure. Supports Lombok annotations with --lombok.",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		workspace, _ := cmd.Flags().GetString("workspace")
		buildFlags, _ := cmd.Flags().GetString("build-flags")
		mode, _ := cmd.Flags().GetString("mode")
		lombok, _ := cmd.Flags().GetBool("lombok")
		packageName, _ := cmd.Flags().GetString("package")
		fileName, _ := cmd.Flags().GetString("output-file")
		if fileName == "" {
			log.Error("File name argument is required")
			return
		}

		// Generate output directory from package name
		outputDir := filepath.Join(strings.ReplaceAll(packageName, ".", string(os.PathSeparator)))
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			log.WithError(err).Errorf("Failed to create output directory: %s", outputDir)
			return
		}

		spinner := utils.NewSpinner("Generating Java types...")

		mgmntClient := utils.GetSuprSendMgmntClient()
		schemasResp, err := mgmntClient.GetLinkedSchemas(cmd.Context(), workspace, mode)
		if err != nil {
			log.WithError(err).Error("Couldn't fetch schemas")
			return
		}

		var schemasToBeGenerated []*schemaToBeGeneratedType
		for _, schema := range schemasResp.Results {
			// loop through schema.LinkedWorkflows and schema.LinkedEvents and add to schemasToBeGenerated
			for _, linkedWorkflow := range schema.LinkedWorkflows {
				schemasToBeGenerated = append(schemasToBeGenerated, &schemaToBeGeneratedType{
					Slug:        schema.Slug,
					VersionNo:   schema.VersionNo,
					Name:        schema.Name,
					TriggerName: CleanTriggerName(linkedWorkflow) + "Workflow",
					JSONSchema:  schema.JSONSchema,
				})
			}
			for _, linkedEvent := range schema.LinkedEvents {
				schemasToBeGenerated = append(schemasToBeGenerated, &schemaToBeGeneratedType{
					Slug:        schema.Slug,
					VersionNo:   schema.VersionNo,
					Name:        schema.Name,
					TriggerName: CleanTriggerName(linkedEvent) + "Event",
					JSONSchema:  schema.JSONSchema,
				})
			}
		}

		if len(schemasToBeGenerated) == 0 {
			spinner.Stop("No valid schemas found")
			log.Warn("No valid schemas found with meaningful JSON schema content")
			return
		}

		generatedCount := 0
		for _, targetSchema := range schemasToBeGenerated {
			schemaName := targetSchema.TriggerName
			javaFileName := filepath.Join(outputDir, schemaName+".java")
			log.Debugf("Processing TargetSchema: %v", targetSchema)

			if _, err := os.Stat(javaFileName); err == nil {
				if err := os.WriteFile(javaFileName, []byte(""), 0o644); err != nil {
					log.WithError(err).Errorf("Failed to clear existing file: %s", javaFileName)
					continue
				}
			}

			schemaJSON := map[string]any{
				"type":       targetSchema.JSONSchema.Type,
				"properties": targetSchema.JSONSchema.Properties,
			}
			if targetSchema.JSONSchema.Required != nil {
				schemaJSON["required"] = targetSchema.JSONSchema.Required
			}
			if targetSchema.JSONSchema.Defs != nil {
				schemaJSON["$defs"] = targetSchema.JSONSchema.Defs
			}
			schemaBytes, err := json.MarshalIndent(schemaJSON, "", "  ")
			if err != nil {
				log.Fatalf("Failed to marshal schema: %v", err)
			}

			javaFlags := buildFlags
			if javaFlags != "" {
				javaFlags += ",just-types=true,package=" + packageName
			} else {
				javaFlags = "just-types=true,package=" + packageName
			}
			if lombok {
				javaFlags += `,lombok="true"`
			}

			err = runTypeMorph("java", string(schemaBytes), schemaName, javaFileName, javaFlags)
			if err != nil {
				log.WithError(err).Errorln("Could not generate types for schema: " + targetSchema.Slug)
			} else {
				generatedCount++
			}
		}
		spinner.Stop(fmt.Sprintf("Generated %d Java type files in %s", generatedCount, outputDir))
	},
}

var generateTypesTypeScriptCmd = &cobra.Command{
	Use:   "typescript [flags]",
	Short: "Generate TypeScript types from JSON Schema",
	Long:  "Generate TypeScript type definitions from trigger payload schemas. Fetches linked schemas and generates types in a single output file. Supports Zod schema generation with --zod.",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		zod, _ := cmd.Flags().GetBool("zod")
		cmd.Flags().Set("build-flags", "just-types=true,prefer-unions=true")
		if zod {
			generateTypesForLanguage("typescript-zod")(cmd, args)
		} else {
			generateTypesForLanguage("typescript")(cmd, args)
		}
	},
}

var generateTypesGoCmd = &cobra.Command{
	Use:   "go [flags]",
	Short: "Generate Go types from JSON Schema",
	Long:  "Generate Go type definitions from trigger payload schemas. Produces struct definitions in a single output file with the specified package name.",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		packageName, _ := cmd.Flags().GetString("package")
		cmd.Flags().Set("build-flags", "just-types-and-package=true,package="+packageName)
		generateTypesForLanguage("go")(cmd, args)
	},
}

var generateTypesKotlinCmd = &cobra.Command{
	Use:   "kotlin [flags]",
	Short: "Generate Kotlin types from JSON Schema",
	Long:  "Generate Kotlin type definitions from trigger payload schemas. Produces data classes in a single output file with the specified package name.",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		packageName, _ := cmd.Flags().GetString("package")
		if packageName != "" {
			cmd.Flags().Set("build-flags", "package="+packageName)
		}
		generateTypesForLanguage("kotlin")(cmd, args)
	},
}

var generateTypesSwiftCmd = &cobra.Command{
	Use:   "swift [flags]",
	Short: "Generate Swift types from JSON Schema",
	Long:  "Generate Swift type definitions from trigger payload schemas. Produces Codable structs in a single output file.",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Flags().Set("build-flags", "coding-keys=true,struct-or-class=struct,initializers=false")
		generateTypesForLanguage("swift")(cmd, args)
	},
}

var generateTypesDartCmd = &cobra.Command{
	Use:   "dart [flags]",
	Short: "Generate Dart types from JSON Schema",
	Long:  "Generate Dart type definitions from trigger payload schemas. Produces null-safe classes in a single output file.",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Flags().Set("build-flags", "just-types=true,null-safety=true")
		generateTypesForLanguage("dart")(cmd, args)
	},
}

func generateTypesForLanguage(targetLang string) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		workspace, _ := cmd.Flags().GetString("workspace")
		buildFlags, _ := cmd.Flags().GetString("build-flags")
		mode, _ := cmd.Flags().GetString("mode")
		fileName, _ := cmd.Flags().GetString("output-file")
		if fileName == "" {
			log.Error("File name argument is required")
			return
		}

		expectedExt := utils.GetExtensionForLanguage(targetLang)
		fileExtension := strings.ToLower(filepath.Ext(fileName))
		if fileExtension != expectedExt {
			log.Errorf("File extension %s doesn't match expected extension %s for %s", fileExtension, expectedExt, targetLang)
			return
		}

		spinner := utils.NewSpinner(fmt.Sprintf("Generating %s types...", cases.Title(language.English).String(targetLang)))

		mgmntClient := utils.GetSuprSendMgmntClient()
		schemasResp, err := mgmntClient.GetLinkedSchemas(cmd.Context(), workspace, mode)
		if err != nil {
			log.WithError(err).Error("Couldn't fetch schemas")
			return
		}

		var schemasToBeGenerated []*schemaToBeGeneratedType
		for _, schema := range schemasResp.Results {
			// loop through schema.LinkedWorkflows and schema.LinkedEvents and add to schemasToBeGenerated
			for _, linkedWorkflow := range schema.LinkedWorkflows {
				schemasToBeGenerated = append(schemasToBeGenerated, &schemaToBeGeneratedType{
					Slug:        schema.Slug,
					VersionNo:   schema.VersionNo,
					Name:        schema.Name,
					TriggerName: CleanTriggerName(linkedWorkflow) + "Workflow",
					JSONSchema:  schema.JSONSchema,
				})
			}
			for _, linkedEvent := range schema.LinkedEvents {
				schemasToBeGenerated = append(schemasToBeGenerated, &schemaToBeGeneratedType{
					Slug:        schema.Slug,
					VersionNo:   schema.VersionNo,
					Name:        schema.Name,
					TriggerName: CleanTriggerName(linkedEvent) + "Event",
					JSONSchema:  schema.JSONSchema,
				})
			}
		}

		if len(schemasToBeGenerated) == 0 {
			spinner.Stop("No valid schemas found")
			log.Warn("No valid schemas found with meaningful JSON schema content")
			return
		}

		if _, err := os.Stat(fileName); err == nil {
			if err := os.WriteFile(fileName, []byte(""), 0o644); err != nil {
				log.WithError(err).Errorf("Failed to clear existing file: %s", fileName)
				return
			}
		}

		generatedCount := 0
		for _, targetSchema := range schemasToBeGenerated {
			schemaName := targetSchema.TriggerName
			log.Debugf("Processing TargetSchema: %v", targetSchema)

			schemaJSON := map[string]any{
				"type":       targetSchema.JSONSchema.Type,
				"properties": targetSchema.JSONSchema.Properties,
			}
			if targetSchema.JSONSchema.Required != nil {
				schemaJSON["required"] = targetSchema.JSONSchema.Required
			}
			if targetSchema.JSONSchema.Defs != nil {
				schemaJSON["$defs"] = targetSchema.JSONSchema.Defs
			}
			schemaBytes, err := json.MarshalIndent(schemaJSON, "", "  ")
			if err != nil {
				log.Fatalf("Failed to marshal schema: %v", err)
			}
			err = runTypeMorph(targetLang, string(schemaBytes), schemaName, fileName, buildFlags)
			if err != nil {
				log.WithError(err).Errorln("Could not generate types for schema: " + targetSchema.Slug)
			} else {
				generatedCount++
			}
		}
		spinner.Stop(fmt.Sprintf("Generated %d %s types in %s", generatedCount, targetLang, fileName))
	}
}

func init() {
	commonFlags := []*cobra.Command{
		generateTypesPythonCmd,
		generateTypesTypeScriptCmd,
		generateTypesGoCmd,
		generateTypesKotlinCmd,
		generateTypesSwiftCmd,
		generateTypesDartCmd,
		generateTypesJavaCmd,
	}

	for _, cmd := range commonFlags {
		cmd.Flags().String("workspace", "staging", "Workspace name (e.g., staging, production)")
		cmd.Flags().String("mode", "live", "Version mode: draft or live")
		cmd.Flags().String("build-flags", "", "Flags to generate types in a certain way.")
		cmd.Flags().MarkHidden("build-flags")
	}
	// Python
	generateTypesPythonCmd.Flags().Bool("pydantic", true, "Generate Pydantic BaseModel classes instead of plain dataclasses")
	generateTypesPythonCmd.Flags().String("output-file", "suprsend_types.py", "Output file path for generated types")
	generateTypesCmd.AddCommand(generateTypesPythonCmd)
	// Java
	generateTypesJavaCmd.Flags().Bool("lombok", false, "Add Lombok annotations to generated classes")
	generateTypesJavaCmd.Flags().String("package", "suprsend.types", "Java package name for generated classes")
	generateTypesJavaCmd.Flags().String("output-file", "SuprsendTypes.java", "Output file path for generated types")
	generateTypesCmd.AddCommand(generateTypesJavaCmd)
	// TypeScript
	generateTypesTypeScriptCmd.Flags().Bool("zod", false, "Generate Zod schemas instead of plain TypeScript types")
	generateTypesTypeScriptCmd.Flags().String("output-file", "suprsend-types.ts", "Output file path for generated types")
	generateTypesCmd.AddCommand(generateTypesTypeScriptCmd)
	// Go
	generateTypesGoCmd.Flags().String("package", "suprsend", "Go package name for generated structs")
	generateTypesGoCmd.Flags().String("output-file", "suprsend_types.go", "Output file path for generated types")
	generateTypesCmd.AddCommand(generateTypesGoCmd)
	// Kotlin
	generateTypesKotlinCmd.Flags().String("package", "suprsend", "Kotlin package name for generated data classes")
	generateTypesKotlinCmd.Flags().String("output-file", "SuprsendTypes.kt", "Output file path for generated types")
	generateTypesCmd.AddCommand(generateTypesKotlinCmd)
	// Swift
	generateTypesSwiftCmd.Flags().String("output-file", "SuprsendTypes.swift", "Output file path for generated types")
	generateTypesCmd.AddCommand(generateTypesSwiftCmd)
	// Dart
	generateTypesDartCmd.Flags().String("output-file", "suprsend_types.dart", "Output file path for generated types")
	generateTypesCmd.AddCommand(generateTypesDartCmd)
	rootCmd.AddCommand(generateTypesCmd)
}

func runTypeMorph(language, schema, schemaName, fileName, buildFlags string) error {
	if len(utils.TypeMorphBin) == 0 {
		return fmt.Errorf("type generation is not supported on %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	binaryPath, err := writeTempExecutable(utils.TypeMorphBin)
	if err != nil {
		return fmt.Errorf("failed to initialize type generator: %w", err)
	}
	defer os.Remove(binaryPath)

	args := []string{language, schema, schemaName, fileName}
	if buildFlags != "" {
		args = append(args, "--build-flags="+buildFlags)
	}

	cmd := exec.Command(binaryPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("type generation failed for %s: %w", fileName, err)
	}
	return nil
}

func writeTempExecutable(data []byte) (string, error) {
	pattern := "typemorph-*"
	if runtime.GOOS == "windows" {
		pattern = "typemorph-*.exe"
	}
	tmpFile, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	if _, err := tmpFile.Write(data); err != nil {
		return "", err
	}

	// Make it executable
	if err := os.Chmod(tmpFile.Name(), 0o755); err != nil {
		return "", err
	}

	return tmpFile.Name(), nil
}

func CleanTriggerName(input string) string {
	// Split on spaces, underscores, or hyphens (1 or more in a row)
	re := regexp.MustCompile(`[ _-]+`)
	parts := re.Split(input, -1)

	var result strings.Builder
	for _, part := range parts {
		if part == "" {
			continue // skip empty parts
		}
		// Lowercase everything after the first char
		first := strings.ToUpper(part[:1])
		rest := ""
		if len(part) > 1 {
			rest = strings.ToLower(part[1:])
		}
		result.WriteString(first + rest)
	}
	return result.String()
}
