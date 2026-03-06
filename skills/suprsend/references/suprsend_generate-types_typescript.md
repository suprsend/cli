# suprsend generate-types typescript

Generate TypeScript types from JSON Schema

```
suprsend generate-types typescript [flags]
```

### Options

```
  -h, --help                 help for typescript
      --mode string          Mode of schema to fetch (draft, live), default: live (default "live")
      --output-file string   Output file for generated TypeScript types (default "suprsend-types.ts")
      --workspace string     Workspace to get schemas from. (default "staging")
      --zod                  Generate Zod types for TypeScript
```

### Options inherited from parent commands

```
      --config string      config file (default: $HOME/.suprsend.yaml)
  -n, --no-color           Disable color output (default: $NO_COLOR)
  -v, --verbosity string   Log level (debug, info, warn, error, fatal, panic) (default "info")
```

