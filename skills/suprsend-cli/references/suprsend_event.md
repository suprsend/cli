# suprsend event

Manage events

Manage events. Subcommands let you list events, get a single event, pull event definitions to local files, and push event-schema mappings to a workspace.

```
suprsend event [flags]
```

## Examples

```
  suprsend event list --output json
  suprsend event get ORDER_RECEIVED
  suprsend event pull --dir ./suprsend/events
```

### Options

```
  -h, --help               help for event
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

