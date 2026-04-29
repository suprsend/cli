# suprsend category get

Get categories and translations

Retrieve preference categories and their translations from a workspace. Returns the full category structure along with translations for all non-English locales. Use --mode to switch between draft and live versions.

```
suprsend category get [flags]
```

## Examples

```
  # Get all categories and translations
  suprsend category get

  # Get draft categories
  suprsend category get --mode draft

  # Get with JSON output
  suprsend category get --output json
```

### Tips

- Use `-o json` for machine-readable JSON output, `-o yaml` for YAML.

### Options

```
  -h, --help            help for get
  -m, --mode string     Version mode: draft or live (default "live")
  -o, --output string   Output format: json or yaml (default "json")
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

