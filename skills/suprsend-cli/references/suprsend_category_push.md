# suprsend category push

Push categories to a workspace

Upload local preference categories and translations to a workspace. Reads `categories.json` and `translations/<locale>.json` from the input directory (default `./suprsend/preference_categories`). Categories are staged as a draft unless --commit is passed. Translation files (every locale except `en`) are pushed first and take effect immediately, with or without --commit.

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

- Categories are written to the **draft** state. Run `suprsend category commit` to promote draft → live, or pass `--commit` to push + commit in one step.
- Translations are not drafted: every non-English `translations/<locale>.json` (or `translations` entry in `--json`) is pushed **live immediately**, with or without `--commit`, before the categories are pushed.
- `--dry-run` makes no API calls: it parses `categories.json` (or `--json`), lists the translation files without reading them, and reports counts. A malformed translation file is only caught by a real push. Nothing is validated server-side.
- The `--commit` confirmation prompt only appears when stdin is a TTY. In non-interactive shells (CI, agents) it proceeds without asking, even without `-F`.
- The result lists each translation locale and then `preference_categories` as `updated` (saved as draft), `unchanged` (the draft already matches `categories.json`, so nothing was sent; with `--commit`, the draft already matched live), `committed` or `failed`. With `--commit` the push and the commit are one step on the server: if the commit is refused (a removed category is still used by an active workflow), exit code 1 and the draft is left as it was. If the draft changes in the workspace while the push runs, it fails with `api-conflict` and nothing is saved; run it again. Locales are `created`, `updated` or `unchanged` the same way, followed by a summary; with `-o json` it is one JSON object with `items` and `counts`.

### Options

```
  -c, --commit                  Promote changes from draft to live after pushing
      --commit-message string   Message describing the changes being committed
  -d, --dir string              Directory containing category files (default: ./suprsend/preference_categories)
  -n, --dry-run                 Report what would be pushed without making any changes
  -F, --force                   Skip confirmation prompt when --commit is set
  -h, --help                    help for push
  -j, --json string             Categories (and optional translations) as a JSON object. Required "categories" key holds the preference category structure. Optional "translations" key maps locale codes to objects with "sections" and "categories" keys, e.g. '{"categories":{"root_categories":[...]},"translations":{"es":{"sections":{"key":{"name":"...","description":"..."}},"categories":{"key":{"name":"...","description":"..."}}}}}'
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

