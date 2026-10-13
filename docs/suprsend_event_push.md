## suprsend event push

Push linked events

### Synopsis

Push event definitions from local per-event directories to a workspace in a single bulk request. Reads `event.json` from every subdirectory of the input directory (default `./suprsend/events`); the event name comes from the `name` field inside each file, not from the directory name.

```
suprsend event push [flags]
```

### Examples

```
  # Push all events from default directory
  suprsend event push

  # Push from a custom directory
  suprsend event push --dir ./my-events

  # Push an event inline via JSON, linked to the "user-signup" schema
  suprsend event push --json '{"events":[{"name":"user_signed_up","payload_schema":{"schema":"user-signup"}}]}'

  # Dry run: preview what would be pushed without making changes
  suprsend event push --dry-run
```

### Options

```
  -d, --dir string    Directory containing per-event subdirectories (default: ./suprsend/events)
  -n, --dry-run       Print what would be pushed without making any changes
  -h, --help          help for push
  -j, --json string   Events payload as a JSON object with an "events" array, e.g. '{"events":[{"name":"user_signed_up","description":"Signup completed","payload_schema":{"schema":"user-signup","version_no":1}}]}'. payload_schema links the event to an existing schema by slug (version_no is optional); it is not an inline JSON Schema
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

### SEE ALSO

* [suprsend event](suprsend_event.md)	 - Manage events

