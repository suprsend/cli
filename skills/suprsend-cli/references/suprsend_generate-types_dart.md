# suprsend generate-types dart

Generate Dart types from JSON Schema

Generate Dart type definitions from trigger payload schemas. Produces null-safe classes in a single output file.

```
suprsend generate-types dart [flags]
```

### Options

```
  -h, --help                 help for dart
      --mode string          Version mode: draft or live (default "live")
      --output-file string   Output file path for generated types (default "suprsend_types.dart")
      --workspace string     Workspace slug, e.g. staging or production (default: $SUPRSEND_WORKSPACE, else the profile's workspace, else staging)
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
      --no-color               Disable color output (default: $NO_COLOR)
  -o, --output string          Output format: pretty, json, or yaml (default "pretty")
  -q, --quiet                  Suppress info/warn output (errors are still shown)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
```

