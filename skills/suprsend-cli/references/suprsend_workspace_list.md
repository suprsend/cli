# suprsend workspace list

List workspaces

List the SuprSend workspaces the current service token can reach: all of them unless --limit is given. Returns workspace name, slug, mode, and description.

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
  -l, --limit int    Maximum number of workspaces to return (default: all)
      --offset int   Number of workspaces to skip for pagination
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

