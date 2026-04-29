# suprsend translation

Manage Translations

Manage template translations. Subcommands let you list, pull, push, and commit translations for notification templates.

```
suprsend translation [flags]
```

## Examples

```
  suprsend translation list
  suprsend translation get --output json
  suprsend translation pull --dir ./suprsend/translations
  suprsend translation push --commit
```

### Options

```
  -h, --help               help for translation
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

