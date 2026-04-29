# suprsend workflow get

Get workflow details

Retrieve detailed information for a specific workflow by its slug. Returns the full workflow definition including nodes, connections, and configuration. Use --mode to switch between draft and live versions.

```
suprsend workflow get [<slug>] [flags]
```

## Examples

```
  # Get a workflow by slug (positional)
  suprsend workflow get welcome

  # Get using the flag form
  suprsend workflow get --slug welcome

  # Get the draft version
  suprsend workflow get welcome --mode draft
```

### Tips

- Use `-o json` for machine-readable JSON output, `-o yaml` for YAML. Default `-o pretty` outputs a human-friendly table.

### Options

```
  -h, --help            help for get
  -m, --mode string     Version mode: draft or live (default "live")
  -o, --output string   Output format: json or yaml (default "json")
  -g, --slug string     Workflow slug
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

