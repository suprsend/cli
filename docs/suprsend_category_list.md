## suprsend category list

List categories

### Synopsis

List notification preference categories in a workspace. Returns a flattened table with root_category, section, category_name, default_preference, and mandatory channels. Use --mode to switch between draft and live.

```
suprsend category list [flags]
```

### Examples

```
  # List all categories (live mode)
  suprsend category list

  # List draft categories
  suprsend category list --mode draft

  # List with JSON output
  suprsend category list --output json
```

### Options

```
  -h, --help          help for list
  -m, --mode string   Version mode: draft or live (default "live")
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
      --no-color               Disable color output (default: $NO_COLOR)
  -o, --output string          Output format: pretty, json, or yaml (default "pretty")
  -q, --quiet                  Suppress info/warn output (errors are still shown)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace name (e.g., staging, production) (default "staging")
```

### SEE ALSO

* [suprsend category](suprsend_category.md)	 - Manage preference categories

