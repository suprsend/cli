## suprsend event get

Get events

### Synopsis

Retrieve all events and their schema mappings from a workspace. Returns event definitions including names, descriptions, and payload schemas.

```
suprsend event get [flags]
```

### Examples

```
  # Get all events (JSON output recommended for event details)
  suprsend event get

  # Get with JSON output
  suprsend event get --output json

  # Get from a specific workspace
  suprsend event get --workspace production
```

### Options

```
  -h, --help            help for get
  -o, --output string   Output format: json or yaml (default "json")
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
      --no-color               Disable color output (default: $NO_COLOR)
  -q, --quiet                  Suppress info/warn output (errors are still shown)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace name (e.g., staging, production) (default "staging")
```

### SEE ALSO

* [suprsend event](suprsend_event.md)	 - Manage events

