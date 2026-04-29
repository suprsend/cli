# suprsend template

Manage templates

Manage notification templates. Templates define the content and structure of notifications across channels (email, SMS, push, in-app, etc.). Subcommands let you list, get details, pull to local files, push from local files, and commit templates.

```
suprsend template [flags]
```

## Examples

```
  suprsend template list
  suprsend template get welcome-email
  suprsend template pull --dir ./suprsend/templates
  suprsend template push welcome-email --commit
```

### Options

```
  -h, --help               help for template
  -w, --workspace string   Workspace to list templates from (default "staging")
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

