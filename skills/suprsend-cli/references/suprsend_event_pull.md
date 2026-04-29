# suprsend event pull

Pull events from workspace to local directory

Download event definitions from a workspace to local files. Saves each event as events/<name>/event.json in the output directory.

```
suprsend event pull [flags]
```

## Examples

```
  # Pull all events to default directory (suprsend/events/)
  suprsend event pull

  # Pull to a custom directory
  suprsend event pull --dir ./my-events

  # Pull from the production workspace
  suprsend event pull --workspace production
```

### Options

```
  -d, --dir string   Directory to save event files to (default: ./suprsend/events)
  -F, --force        Skip directory confirmation prompt, use default path
  -h, --help         help for pull
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

