# suprsend workspace

Manage workspaces

Manage SuprSend workspaces. Workspaces isolate notification resources (templates, workflows, categories) and can run in sandbox or live mode.

```
suprsend workspace [flags]
```

## Examples

```
  suprsend workspace list
  suprsend workspace list --limit 5
  suprsend workspace list --output json
```

### Options

```
  -h, --help   help for workspace
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

