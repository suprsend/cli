# suprsend schema

Manage trigger payload schemas

Manage trigger payload schemas. Schemas define the JSON structure for workflow and event trigger payloads. Subcommands let you list, pull, push, and commit schemas.

```
suprsend schema [flags]
```

## Examples

```
  suprsend schema list
  suprsend schema get order-placed
  suprsend schema pull --dir ./suprsend/schemas
  suprsend schema push order-placed --commit
```

### Options

```
  -h, --help               help for schema
  -w, --workspace string   Workspace name (e.g., staging, production) (default "staging")
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

