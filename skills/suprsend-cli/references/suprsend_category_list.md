# suprsend category list

List categories

List notification preference categories in a workspace. Returns a flattened table with root_category, section, category_name, default_preference, and mandatory channels. Use --mode to switch between draft and live.

```
suprsend category list [flags]
```

## Examples

```
  # List all categories (live mode)
  suprsend category list

  # List draft categories
  suprsend category list --mode draft

  # List with JSON output
  suprsend category list --output json
```

### Tips

- Use `-o json` for machine-readable JSON output, `-o yaml` for YAML. Default `-o pretty` outputs a human-friendly table.

### Options

```
  -h, --help            help for list
  -m, --mode string     Version mode: draft or live (default "live")
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

