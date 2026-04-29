# suprsend workflow list

List workflows for a workspace

List workflows in a workspace with pagination. Returns workflow slug, name, status, and version info. Use --mode to switch between draft and live versions.

```
suprsend workflow list [flags]
```

## Examples

```
  # List all workflows (live mode)
  suprsend workflow list

  # List draft workflows
  suprsend workflow list --mode draft

  # Paginate with JSON output
  suprsend workflow list --limit 50 --offset 50 --output json
```

### Tips

- Use `-o json` for machine-readable JSON output, `-o yaml` for YAML. Default `-o pretty` outputs a human-friendly table.

### Options

```
  -h, --help            help for list
  -l, --limit int       Maximum number of workflows to return (default 20)
  -m, --mode string     Version mode: draft or live (default "live")
      --offset int      Number of workflows to skip for pagination
  -o, --output string   Output format: pretty, json, or yaml (default "pretty")
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

