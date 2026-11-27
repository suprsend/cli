# suprsend schema list

List schemas

List the trigger payload schemas in a workspace: all of them unless --limit is given. Returns schema slug, name, and version info. Use --mode to switch between draft and live versions.

```
suprsend schema list [flags]
```

## Examples

```
  # List all schemas (live mode)
  suprsend schema list

  # List draft schemas
  suprsend schema list --mode draft

  # Paginate with JSON output
  suprsend schema list --limit 50 --output json
```

### Tips

- Use `-o json` for machine-readable JSON output, `-o yaml` for YAML. Default `-o pretty` outputs a human-friendly table.

### Options

```
  -h, --help            help for list
  -l, --limit int       Maximum number of schemas to return (default: all)
  -m, --mode string     Version mode: draft or live (default "live")
      --offset int      Number of schemas to skip for pagination
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

