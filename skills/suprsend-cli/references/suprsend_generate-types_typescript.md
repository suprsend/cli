# suprsend generate-types typescript

Generate TypeScript types from JSON Schema

Generate TypeScript type definitions from trigger payload schemas. Fetches linked schemas and generates types in a single output file. Supports Zod schema generation with --zod.

```
suprsend generate-types typescript [flags]
```

### Options

```
  -h, --help                 help for typescript
      --mode string          Version mode: draft or live (default "live")
      --output-file string   Output file path for generated types (default "suprsend-types.ts")
      --workspace string     Workspace name (e.g., staging, production) (default "staging")
      --zod                  Generate Zod schemas instead of plain TypeScript types
```

### Options inherited from parent commands

```
      --config string      config file (default: $HOME/.suprsend.yaml)
  -n, --no-color           Disable color output (default: $NO_COLOR)
  -v, --verbosity string   Log level (debug, info, warn, error, fatal, panic) (default "info")
```

