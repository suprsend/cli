# suprsend template list

List templates for a workspace

List templates in a workspace with pagination. Returns template slug, name, and enabled channel info. Use --mode to switch between draft and live versions.

```
suprsend template list [flags]
```

## Examples

```
  # List all templates (live mode)
  suprsend template list

  # List draft templates
  suprsend template list --mode draft

  # Paginate with JSON output
  suprsend template list --limit 50 --output json
```

### Options

```
  -h, --help            help for list
  -l, --limit int       Limit the number of templates to list (default 20)
  -m, --mode string     Version mode: draft or live (default "live")
      --offset int      Offset the number of templates to list (default: 0)
  -o, --output string   Output Style (pretty, yaml, json) (default "pretty")
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
      --no-color               Disable color output (default: $NO_COLOR)
  -q, --quiet                  Suppress info/warn output (errors are still shown)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace to list templates from (default "staging")
```

