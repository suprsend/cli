# suprsend workflow disable

Disable a workflow

Disable a workflow to stop it from processing triggers. Pass the workflow slug as a positional argument or via --slug.

```
suprsend workflow disable [<slug>] [flags]
```

## Examples

```
  # Disable a workflow (prompts for confirmation)
  suprsend workflow disable welcome

  # Disable without confirmation prompt
  suprsend workflow disable welcome --force

  # Dry run: see what would change without making changes
  suprsend workflow disable welcome --dry-run
```

### Options

```
  -n, --dry-run       Print what would be changed without making any changes
  -F, --force         Skip confirmation prompt
  -h, --help          help for disable
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

