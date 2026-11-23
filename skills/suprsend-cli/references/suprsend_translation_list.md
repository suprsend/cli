# suprsend translation list

List translations

List the template translation files in a workspace: all of them unless --limit is given. Returns translation file names and metadata. Use --mode to switch between draft and live versions.

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
  -l, --limit int                Maximum number of translation files to return (default: all)
  -m, --mode string              Version mode: draft or live (default "live")
      --offset int               Number of translations to skip for pagination
  -o, --output string            Output format: pretty, json, or yaml (default "pretty")
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

