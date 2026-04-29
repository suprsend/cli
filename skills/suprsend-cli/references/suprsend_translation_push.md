# suprsend translation push

Push translation files to a workspace

Upload local template translation JSON files to a workspace. Reads all .json files from the input directory and pushes them. Use --commit to also finalize the changes immediately.

```
suprsend translation push [flags]
```

## Examples

```
  # Push translations from default directory
  suprsend translation push

  # Push and commit to live immediately
  suprsend translation push --commit

  # Dry run: preview what would be pushed without making changes
  suprsend translation push --dry-run
```

### Options

```
  -c, --commit                  Promote changes from draft to live after pushing
      --commit-message string   Message describing the changes being committed
  -d, --dir string              Directory containing translation JSON files (default: ./suprsend/translations)
  -n, --dry-run                 Print what would be pushed without making any changes
  -h, --help                    help for push
  -j, --json string             Translations as a JSON object mapping locale codes (without .json extension) to their translation content objects, e.g. '{"en":{"key":"value"},"fr":{"key":"valeur"}}'
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

