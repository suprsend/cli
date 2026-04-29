# suprsend category translation push

Push preference translations

Upload local preference category translation files to a workspace. Reads {locale}.json files from the input directory. English (en.json) is skipped — the API rejects pushes to the source-of-truth locale. Use --locale to push a single locale, or omit to push all.

```
suprsend category translation push [flags]
```

## Examples

```
  # Push all locale translations from default directory
  suprsend category translation push

  # Push a specific locale only
  suprsend category translation push --locale es

  # Push from a custom directory
  suprsend category translation push --dir ./my-categories

  # Dry run: preview what would be pushed without making changes
  suprsend category translation push --dry-run
```

### Options

```
  -d, --dir string      Directory containing translation JSON files (default: suprsend/preference_categories/translations)
  -n, --dry-run         Print what would be pushed without making any changes
  -h, --help            help for push
      --locale string   Locale code to push, e.g., es, fr (omit to push all)
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

