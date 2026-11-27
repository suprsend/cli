# suprsend category list

List categories

List notification preference categories in a workspace. Returns a flattened table with root_category, section, category (the slug that workflows, preferences and translation files use), category_name, default_preference, mandatory channels, and is_commercial. Use --mode to switch between draft and live.

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
      --profile name           Profile to use, by name (default: $SUPRSEND_PROFILE, else the config file's active profile)
  -q, --quiet                  Suppress info and warning output (results and errors are still shown)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace slug, e.g. staging or production (default: $SUPRSEND_WORKSPACE, else the profile's workspace, else staging)
```

