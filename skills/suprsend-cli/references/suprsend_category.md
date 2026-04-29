# suprsend category

Manage preference categories

Manage notification preference categories. Categories organize notification preferences into a hierarchy of root categories, sections, and individual preference items.

```
suprsend category [flags]
```

## Examples

```
  suprsend category list
  suprsend category get --output json
  suprsend category pull --dir ./suprsend/categories
  suprsend category push --commit
```

### Options

```
  -h, --help               help for category
  -w, --workspace string   Workspace name (e.g., staging, production) (default "staging")
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
      --no-color               Disable color output (default: $NO_COLOR)
  -o, --output string          Output format: pretty, json, or yaml (default "pretty")
  -q, --quiet                  Suppress info/warn output (errors are still shown)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
```

