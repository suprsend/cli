# suprsend translation list

List translations

List template translation files in a workspace. Returns translation file names and metadata. Use --mode to switch between draft and live versions.

```
suprsend translation list [flags]
```

## Examples

```
  # List all translations (live mode)
  suprsend translation list

  # List draft translations
  suprsend translation list --mode draft

  # List with JSON output
  suprsend translation list --output json
```

### Tips

- Use `-o json` for machine-readable JSON output, `-o yaml` for YAML. Default `-o pretty` outputs a human-friendly table.

### Options

```
  -h, --help                     help for list
      --include-content string   Include translation file content in the response (true/false) (default "false")
  -l, --limit int                Maximum number of translations to return (default 20)
  -m, --mode string              Version mode: draft or live (default "live")
      --offset int               Number of translations to skip for pagination
  -o, --output string            Output format: pretty, json, or yaml (default "pretty")
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

