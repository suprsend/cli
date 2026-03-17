# suprsend event push

Push linked events

Push event-to-schema mappings from a local event_schema_mapping.json file to a workspace. Reads the mapping file from the specified directory.

```
suprsend event push [flags]
```

### Options

```
  -d, --dir string    Directory containing event files (default: ./suprsend/event)
  -h, --help          help for push
  -j, --json string   Events payload as a JSON object with an "events" array, matching the format produced by pull, e.g. '{"events":[{"name":"user_signed_up","description":"...","payload_schema":{...}}]}'
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
  -n, --no-color               Disable color output (default: $NO_COLOR)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace name (e.g., staging, production) (default "staging")
```

