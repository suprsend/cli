# suprsend translation commit

Commit translation

Promote template translation changes from draft to live mode. Finalizes all pending translation changes in the workspace.

```
suprsend translation commit [flags]
```

## Examples

```
  # Commit all pending translation changes to live
  suprsend translation commit

  # Commit in the production workspace
  suprsend translation commit --workspace production

  # Dry run: see what would be committed without making changes
  suprsend translation commit --dry-run
```

### Options

```
      --commit-message string   Message describing the changes being committed
  -n, --dry-run                 Print what would be committed without making any changes
  -F, --force                   Skip confirmation prompt
  -h, --help                    help for commit
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

