# suprsend category translation

Manage preference category translations

Manage preference category translations. List available translation locales, pull translations from a workspace to local files, or push local translation files back to a workspace.

```
suprsend category translation [flags]
```

## Examples

```
  suprsend category translation list
  suprsend category translation pull --dir ./suprsend/categories
  suprsend category translation push --locale es
```

### Options

```
  -h, --help   help for translation
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

