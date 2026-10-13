# suprsend event pull

Pull events from workspace to local directory

Download event definitions from a workspace to local files. Saves each event as `<dir>/<event_dir>/event.json` (default dir `./suprsend/events`). `<event_dir>` is the event name made filesystem-safe: characters other than letters, digits, `.`, `_` and `-` become `_`, leading and trailing non-alphanumeric characters are dropped (a leading `$` is kept), and the result is capped at 128 characters. If two event names map to the same directory name (compared case-insensitively), only the first is written and the other is skipped with a warning.

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

### Tips

- Pull overwrites `<dir>/<event_dir>/event.json` for every pulled event without asking, so commit or stash local edits first. `--force` does not affect overwriting: it only skips the output-directory prompt shown when `--dir` is omitted and `./suprsend/events` doesn't exist. In non-interactive shells pass `--dir` or `--force`, otherwise nothing is pulled.
- The directory name is a sanitized form of the event name (e.g. `Order Placed` → `Order_Placed`), so don't derive the event name from it: the real name is the `name` field inside `event.json`.

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

