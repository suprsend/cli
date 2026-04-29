# suprsend event push

Push linked events

Push event definitions from local per-event directories to a workspace. Reads from events/<name>/event.json files in the specified directory.

```
suprsend event push [flags]
```

## Examples

```
  # Push all events from default directory
  suprsend event push

  # Push from a custom directory
  suprsend event push --dir ./my-events

  # Push events inline via JSON
  suprsend event push --json '{"events":[{"name":"user_signed_up","payload_schema":{...}}]}'

  # Dry run: preview what would be pushed without making changes
  suprsend event push --dry-run
```

### Options

```
  -d, --dir string    Directory containing per-event subdirectories (default: ./suprsend/events)
  -n, --dry-run       Print what would be pushed without making any changes
  -h, --help          help for push
  -j, --json string   Events payload as a JSON object with an "events" array, e.g. '{"events":[{"name":"user_signed_up","description":"...","payload_schema":{...}}]}'
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

