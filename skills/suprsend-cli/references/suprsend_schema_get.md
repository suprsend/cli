# suprsend schema get

Get schema details

Retrieve the full definition of a specific schema by its slug. Returns the JSON Schema object including type, properties, and validation rules. Use --mode to switch between draft and live versions.

```
suprsend schema get [<slug>] [flags]
```

## Examples

```
  # Get a schema by slug (positional)
  suprsend schema get order-placed

  # Get using the flag form
  suprsend schema get --slug order-placed

  # Get the draft version
  suprsend schema get order-placed --mode draft
```

### Tips

- Use `-o json` for machine-readable JSON output, `-o yaml` for YAML.

### Options

```
  -h, --help            help for get
  -m, --mode string     Version mode: draft or live (default "live")
  -o, --output string   Output format: json or yaml (default "json")
  -g, --slug string     Schema slug
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
      --no-color               Disable color output (default: $NO_COLOR)
  -q, --quiet                  Suppress info/warn output (errors are still shown)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace name (e.g., staging, production) (default "staging")
```

