# suprsend event get

Get event details

Retrieve a specific event by its slug. Returns the event definition including name, description, and payload schema.

```
suprsend event get [<slug>] [flags]
```

## Examples

```
  # Get a specific event by slug
  suprsend event get ORDER_RECEIVED

  # Get using the flag form
  suprsend event get --slug ORDER_RECEIVED

  # Get from a specific workspace
  suprsend event get --workspace production
```

### Tips

- Use `-o json` for machine-readable JSON output, `-o yaml` for YAML.

### Options

```
  -h, --help            help for get
  -o, --output string   Output format: json or yaml (default "json")
  -g, --slug string     Event slug
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

