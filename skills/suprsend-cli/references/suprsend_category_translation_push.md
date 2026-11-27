# suprsend category translation push

Push preference translations

Upload local preference category translation files to a workspace. Reads {locale}.json files directly from the input directory (default `./suprsend/preference_categories/translations`). English (en.json) is skipped — the API rejects pushes to the source-of-truth locale. Translations have no draft state: pushed files take effect immediately. Use --locale to push a single locale, or omit to push all.

```
suprsend category translation push [flags]
```

## Examples

```
  # Push all locale translations from default directory
  suprsend category translation push

  # Push a specific locale only
  suprsend category translation push --locale es

  # Push from a custom directory (the directory that contains the locale files)
  suprsend category translation push --dir ./my-categories/translations

  # Dry run: preview what would be pushed without making changes
  suprsend category translation push --dry-run
```

### Tips

- Each locale is compared with the workspace's and reported on stdout as `created` or `updated` (translations go live immediately), `unchanged` (the workspace already has the same, so nothing was sent), `skipped` (English) or `failed`, followed by a summary. Keys the API didn't save (not a section or category slug in the workspace) are named in the locale's result line: fix or remove them, since the locale is otherwise pushed again every time; with `-o json` the result is one JSON object with `items` and `counts`.

### Options

```
  -d, --dir string      Directory containing translation JSON files (default: suprsend/preference_categories/translations)
  -n, --dry-run         Report what would be pushed without making any changes
  -h, --help            help for push
      --locale string   Locale code to push, e.g., es, fr (omit to push all)
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
      --no-color               Disable color output (default: $NO_COLOR)
  -o, --output string          Output format: pretty, json, or yaml (default "pretty")
      --profile name           Profile to use, by name (default: $SUPRSEND_PROFILE, else the config file's active profile)
  -q, --quiet                  Suppress info and warning output (results and errors are still shown)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace slug, e.g. staging or production (default: $SUPRSEND_WORKSPACE, else the profile's workspace, else staging)
```

