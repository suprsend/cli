# suprsend workflow commit

Commit workflow from draft to live

Promote a workflow from draft to live mode. Pass the workflow slug as a positional argument or via --slug. Once committed, the workflow changes become active immediately.

```
suprsend workflow commit [<slug>] [flags]
```

## Examples

```
  # Commit a workflow to live (positional slug)
  suprsend workflow commit welcome

  # Commit using the flag form
  suprsend workflow commit --slug welcome

  # Dry run: see what would be committed without making changes
  suprsend workflow commit welcome --dry-run
```

### Tips

- Commit is irreversible: it promotes the draft to **live**, and live workflows immediately begin executing the new definition for new trigger events.
- If you didn't author the draft locally, run `suprsend workflow get --slug <slug> --mode draft` first to inspect what will become live.

### Options

```
      --commit-message string   Message describing the changes being committed
  -n, --dry-run                 Print what would be committed without making any changes
  -F, --force                   Skip confirmation prompt
  -h, --help                    help for commit
  -g, --slug string             Workflow slug
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

