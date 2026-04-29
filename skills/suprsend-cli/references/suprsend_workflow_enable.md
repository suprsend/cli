# suprsend workflow enable

Enable a workflow

Enable a workflow to make it active and ready to receive triggers. Pass the workflow slug as a positional argument or via --slug.

```
suprsend workflow enable [<slug>] [flags]
```

## Examples

```
  # Enable a workflow by slug (positional)
  suprsend workflow enable welcome

  # Enable using the flag form
  suprsend workflow enable --slug welcome

  # Dry run: see what would change without making changes
  suprsend workflow enable welcome --dry-run
```

### Options

```
  -n, --dry-run       Print what would be changed without making any changes
  -h, --help          help for enable
  -g, --slug string   Workflow slug
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

