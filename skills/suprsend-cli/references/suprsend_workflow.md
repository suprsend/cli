# suprsend workflow

Manage workflows

Manage workflows. Subcommands let you list, get details, pull to local files, push from local files, and enable/disable workflows in a workspace.

```
suprsend workflow [flags]
```

## Examples

```
  suprsend workflow list
  suprsend workflow get welcome
  suprsend workflow pull --dir ./suprsend/workflows
  suprsend workflow push welcome --commit
```

### Options

```
  -h, --help               help for workflow
  -w, --workspace string   Workspace name (e.g., staging, production) (default "staging")
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

