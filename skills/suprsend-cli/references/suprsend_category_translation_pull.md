# suprsend category translation pull

Pull preference translations

Download preference category translations from a workspace to local JSON files. Creates one file per locale (e.g., es.json, fr.json) in the output directory.

```
suprsend category translation pull [flags]
```

## Examples

```
  # Pull all locale translations to default directory
  suprsend category translation pull

  # Pull to a custom directory
  suprsend category translation pull --dir ./my-categories

  # Pull in the production workspace
  suprsend category translation pull --workspace production
```

### Options

```
  -d, --dir string   Directory to save translation files to (default: suprsend/preference_categories/translations)
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

