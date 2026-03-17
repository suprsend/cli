# suprsend event pull

Pull events from workspace to local directory

Download event definitions from a workspace to local files. Saves event data and an event_schema_mapping.json file to the output directory.

```
suprsend event pull [flags]
```

### Options

```
  -d, --dir string   Directory to save event files to (default: ./suprsend/event)
  -f, --force        Skip directory confirmation prompt, use default path
  -h, --help         help for pull
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
  -n, --no-color               Disable color output (default: $NO_COLOR)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace name (e.g., staging, production) (default "staging")
```

