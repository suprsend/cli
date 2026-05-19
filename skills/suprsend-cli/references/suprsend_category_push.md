# suprsend category push

Push categories to a workspace

Upload local preference categories and translations to a workspace. Reads categories_preferences.json and translation files from the input directory. By default, changes are staged as drafts. Use --commit to also promote to live.

```
suprsend category push [flags]
```

## Examples

```
  # Push from local files (default directory)
  suprsend category push

  # Push from a custom directory
  suprsend category push --dir ./my-categories

  # Push and commit to live immediately
  suprsend category push --commit

  # Push categories inline via JSON
  suprsend category push --json '{"categories": {...}}'
```

### Tips

- Push writes to the **draft** state. Run `suprsend category commit` to promote draft → live.
- Pair with `--dry-run` to validate the categories server-side without writing to the draft. Pair with `--commit` to push + commit in one step.

### Options

```
  -c, --commit                  Promote changes from draft to live after pushing
      --commit-message string   Message describing the changes being committed
  -d, --dir string              Directory containing category files (default: ./suprsend/preference_categories)
  -n, --dry-run                 Print what would be pushed without making any changes
  -h, --help                    help for push
  -j, --json string             Categories (and optional translations) as a JSON object. Required "categories" key holds the preference category structure. Optional "translations" key maps locale codes to objects with "sections" and "categories" keys, e.g. '{"categories":{"root_categories":[...]},"translations":{"es":{"sections":{"key":{"name":"...","description":"..."}},"categories":{"key":{"name":"...","description":"..."}}}}}'
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

