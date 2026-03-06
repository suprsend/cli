# suprsend event list

List events

List all events

```
suprsend event list [flags]
```

### Options

```
  -h, --help            help for list
  -l, --limit int       Limit the number of events to list. (default 20)
  -f, --offset int      Offset into the list of events(default: 0)
  -o, --output string   Output Style (pretty, yaml, json) (default "pretty")
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
  -n, --no-color               Disable color output (default: $NO_COLOR)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace to list events from (default "staging")
```

