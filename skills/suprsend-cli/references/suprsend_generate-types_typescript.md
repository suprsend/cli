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
  -w, --workspace string     Workspace slug, e.g. staging or production (default: $SUPRSEND_WORKSPACE, else the profile's workspace, else staging)
      --zod                  Generate Zod schemas instead of plain TypeScript types
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
      --no-color               Disable color output (default: $NO_COLOR)
  -o, --output string          Output format: pretty, json, or yaml (default "pretty")
      --profile name           Profile to use, by name (default: $SUPRSEND_PROFILE, else the config file's active profile)
  -q, --quiet                  Suppress info and warning output (results and errors are still shown)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
```

