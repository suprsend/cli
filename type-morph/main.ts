// deno-lint-ignore-file
import {
  quicktype,
  quicktypeMultiFile,
  JSONSchemaInput,
  FetchingJSONSchemaStore,
  InputData,
} from "npm:quicktype-core@23.2.6";
import $RefParser from "npm:@apidevtools/json-schema-ref-parser";
import { cleanAndHoistImports, Lang } from './import-cleaner.ts';


async function quicktypeJSONSchema(
  targetLanguage: any,
  typeName: string,
  jsonSchemaString: string,
  rendererOptions: Record<string, string> = {},
) {
  const schemaInput = new JSONSchemaInput(new FetchingJSONSchemaStore());
  let flattenedSchema;
  try {
    const dereferencedSchema = await $RefParser.dereference(JSON.parse(jsonSchemaString), {
      dereference: { circular: "ignore" },
    });
    flattenedSchema = dereferencedSchema
  } catch (error) {
    console.error("Error parsing or dereferencing schema:", error);
    throw new Error("Invalid JSON schema or failed to resolve references");
  }
  await schemaInput.addSource({
    name: typeName,
    schema: JSON.stringify(flattenedSchema),
  });
  const inputData = new InputData();
  inputData.addInput(schemaInput);
  const renderOptions = {
    "just-types-package": "true",
    "no-extra-properties": "true",
    "no-optional-null": "true",
    "framework": "just-types",
    ...rendererOptions
  };
  if (targetLanguage.toLowerCase() === 'java') {
    const result = await quicktypeMultiFile({
      inputData,
      lang: targetLanguage,
      rendererOptions: renderOptions
    });
    const schema: Record<string, string[]> = {};
    result.forEach((content, fileName) => {
      schema[fileName] = content.lines
    });
    return schema
  }
  return await quicktype({
    inputData,
    lang: targetLanguage,
    rendererOptions: renderOptions
  });
}

function extractRendererOptions(args: string[]): Record<string, string> {
  const options: Record<string, string> = {};
  for (const arg of args) {
    if (arg.startsWith("--build-flags=")) {
      const flags = arg.substring("--build-flags=".length);
      for (const flag of flags.split(",")) {
        const trimmed = flag.trim();
        if (trimmed) {
          if (trimmed.includes("=")) {
            const equalIndex = trimmed.indexOf("=");
            const key = trimmed.substring(0, equalIndex);
            const value = trimmed.substring(equalIndex + 1);
            options[key] = value;
          } else {
            options[trimmed] = "true";
          }
        }
      }
    }
  }
  return options;
}

async function main() {
  const [
    language,
    schemaInput,
    schemaName,
    outputPath,
    ...extraArgs
  ] = Deno.args;

  let text: string;
  const rendererOptions = extractRendererOptions(extraArgs);
  try {
    text = await Deno.readTextFile(schemaInput);
  } catch (_) {
    try {
      JSON.parse(schemaInput);
      text = schemaInput;
    } catch (err) {
      console.error("Error reading schema input:", err);
      Deno.exit(1);
    }
  }
  let schema: Record<string, any>;
  try {
    schema = JSON.parse(text);
  } catch (err) {
    console.error("Error parsing schema JSON:", err);
    Deno.exit(1);
  }
  //log all inputs in a single line
  console.log("Inputs: language=", language, "schemaInput=", schemaInput, "schemaName=", schemaName, "outputPath=", outputPath, "rendererOptions=", rendererOptions);
  const transformedSchema = transformSchema({
    schema: { ...schema },
    parentTitle: schemaName + "Data",
    title: schemaName.replace(/(Event|Workflow)$/, "") 
  });
  const transformedText = JSON.stringify(transformedSchema, null, 2);
  const rootTypeName = transformedSchema.title || schemaName + "Data";
  let output;
  if (language.toLowerCase() === "java") {
    const schema = await quicktypeJSONSchema(language, rootTypeName, transformedText, rendererOptions);
    for (const [fileName, content] of Object.entries(schema)) {
      output = content.join("\n");
      const directory = outputPath.replace(/[^/\\]+$/, '');
      await writeGeneratedTypes(output, language, directory + fileName, rendererOptions, true);
    }
  } else {
    const { lines } = await quicktypeJSONSchema(language, rootTypeName, transformedText, rendererOptions);
    output = lines.join("\n");
    await writeGeneratedTypes(output, language, outputPath, rendererOptions);
  }
}

async function writeGeneratedTypes(
  output: string,
  language: string,
  outputPath: string,
  rendererOptions: Record<string, string> = {},
  forceOverwrite: boolean = false
): Promise<void> {
  const directory = outputPath.substring(0, outputPath.lastIndexOf('/'));
  if (directory && directory !== outputPath) {
    try {
      await Deno.mkdir(directory, { recursive: true });
    } catch (_) {
    }
  }

  const getCommentPrefix = (lang: string) => {
    switch (lang.toLowerCase()) {
      case 'python':
        return '#';
      default:
        return '//';
    }
  };
  const commentPrefix = getCommentPrefix(language);
  const currentDateTime = new Date().toISOString().replace('T', ' ').replace(/\.\d{3}Z$/, ' UTC');
  let packageDeclaration = "";
  if (language.toLowerCase() === 'java' && rendererOptions.package) {
    packageDeclaration = "";
  }
  const autoGeneratedComment = `${commentPrefix} This file is generated automatically by SuprSend based on available schema. DO NOT MODIFY IT.\n${commentPrefix} Generated on ${currentDateTime}\n\n`;
  let existingContent = "";
  if (!forceOverwrite) {
    try {
      existingContent = await Deno.readTextFile(outputPath);
    } catch (_) { }
  }

  if (existingContent === "" || forceOverwrite) {
    let finalOutput = autoGeneratedComment + packageDeclaration + output;
    finalOutput = cleanAndHoistImports(finalOutput, language.toLowerCase() as Lang);
    await Deno.writeTextFile(outputPath, finalOutput);
  } else {
    let finalOutput = existingContent + "\n\n" + output;
    finalOutput = cleanAndHoistImports(finalOutput, language.toLowerCase() as Lang);
    await Deno.writeTextFile(outputPath, finalOutput);
  }
}

function cleanupEmptyLines(content: string): string {
  const lines = content.split('\n');
  const cleanedLines: string[] = [];
  let consecutiveEmptyLines = 0;
  for (const line of lines) {
    if (line.trim() === '') {
      consecutiveEmptyLines++;
      if (consecutiveEmptyLines <= 1) {
        cleanedLines.push(line);
      }
    } else {
      consecutiveEmptyLines = 0;
      cleanedLines.push(line);
    }
  }
  while (cleanedLines.length > 0 && cleanedLines[cleanedLines.length - 1].trim() === '') {
    cleanedLines.pop();
  }
  return cleanedLines.join('\n');
}


interface TransformSchemaArgs {
  schema: Record<string, any>;
  title?: string;                  // prefix for root's immediate children
  parentTitle?: string;            // override ONLY for the root's type name
  renameMap?: Record<string, string>;
  isRoot?: boolean;
}

export function transformSchema({
  schema,
  title,
  parentTitle,
  renameMap = {},
  isRoot = true,
}: TransformSchemaArgs): Record<string, any> {
  // Capture the original title before we touch it
  const originalRootTitle = isRoot ? (schema.title as string | undefined) : undefined;

  // Root renaming with parentTitle (applies only once)
  if (isRoot && parentTitle) {
    if (schema.title && schema.title !== parentTitle) {
      renameMap[schema.title] = parentTitle;
    }
    schema.title = parentTitle;
  }

  // Close objects by default
  if (schema.type === "object" && schema.additionalProperties === undefined) {
    schema.additionalProperties = false;
  }

  // Prefix for this node’s children
  const prefixForChildren: string | undefined =
    isRoot ? (title ?? originalRootTitle) : (schema.title as string | undefined);

  // Handle object properties
  if (schema.type === "object" && schema.properties && typeof schema.properties === "object") {
    for (const key of Object.keys(schema.properties)) {
      const child = schema.properties[key];
      const baseChildTitle = child.title ?? capitalize(key);
      const newChildTitle = prefixForChildren ? `${prefixForChildren}${baseChildTitle}` : baseChildTitle;

      if (child.title !== newChildTitle) {
        if (child.title) renameMap[child.title] = newChildTitle;
        child.title = newChildTitle;
      }

      schema.properties[key] = transformSchema({
        schema: child,
        renameMap,
        isRoot: false,
      });

      fixRefsDeep(schema.properties[key], renameMap);
    }
  }

  // Handle arrays
  if (schema.type === "array" && schema.items) {
    const items = schema.items;
    const baseItemsTitle = items.title ?? "Item";
    const newItemsTitle = prefixForChildren ? `${prefixForChildren}${baseItemsTitle}` : baseItemsTitle;

    if (items.title !== newItemsTitle) {
      if (items.title) renameMap[items.title] = newItemsTitle;
      items.title = newItemsTitle;
    }

    schema.items = transformSchema({
      schema: items,
      renameMap,
      isRoot: false,
    });

    fixRefsDeep(schema.items, renameMap);
  }

  // Handle allOf / anyOf / oneOf
  for (const combiner of ["allOf", "anyOf", "oneOf"] as const) {
    if (Array.isArray((schema as any)[combiner])) {
      (schema as any)[combiner] = (schema as any)[combiner].map((sub: any) =>
        transformSchema({
          schema: sub,
          renameMap,
          isRoot: false,
        })
      );
    }
  }

  // Handle $defs
  if (schema.$defs && typeof schema.$defs === "object") {
    for (const defKey of Object.keys(schema.$defs)) {
      const defSchema = schema.$defs[defKey];
      const baseDefTitle = defSchema.title ?? capitalize(defKey);
      const newDefTitle = prefixForChildren ? `${prefixForChildren}${baseDefTitle}` : baseDefTitle;

      if (defSchema.title !== newDefTitle) {
        if (defSchema.title) renameMap[defSchema.title] = newDefTitle;
        defSchema.title = newDefTitle;
      }

      schema.$defs[defKey] = transformSchema({
        schema: defSchema,
        renameMap,
        isRoot: false,
      });

      fixRefsDeep(schema.$defs[defKey], renameMap);
    }
  }

  fixRefHere(schema, renameMap);

  return schema;
}

function fixRefHere(s: Record<string, any>, renameMap: Record<string, string>) {
  if (s.$ref && typeof s.$ref === "string" && s.$ref.startsWith("#/$defs/")) {
    const refName = s.$ref.slice("#/$defs/".length);
    if (renameMap[refName]) {
      s.$ref = `#/$defs/${renameMap[refName]}`;
    }
  }
}

function fixRefsDeep(s: any, renameMap: Record<string, string>) {
  if (!s || typeof s !== "object") return;
  fixRefHere(s, renameMap);
  for (const k of Object.keys(s)) {
    const v = (s as any)[k];
    if (v && typeof v === "object") fixRefsDeep(v, renameMap);
  }
}

function capitalize(str: string): string {
  return str ? str.charAt(0).toUpperCase() + str.slice(1) : str;
}


main();