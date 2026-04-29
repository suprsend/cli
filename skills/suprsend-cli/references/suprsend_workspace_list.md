# suprsend workspace list

List workspaces

List all SuprSend workspaces accessible with the current service token. Returns workspace name, slug, mode, and description.

```
suprsend workspace list [flags]
```

## Examples

```
  # List workspaces
  suprsend workspace list

  # Paginate results
  suprsend workspace list --limit 5 --offset 0

  # JSON output
  suprsend workspace list --output json
```

### Tips

- Use `-o json` for machine-readable JSON output, `-o yaml` for YAML. Default `-o pretty` outputs a human-friendly table.

### Options

```
  -h, --help         help for list
  -l, --limit int    Maximum number of workspaces to return (default 20)
      --offset int   Number of workspaces to skip for pagination
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

