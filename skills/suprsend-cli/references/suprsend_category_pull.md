# suprsend category pull

Pull categories from a workspace

Download preference categories and their translations from a workspace to local files. Writes `categories.json` and one `translations/<locale>.json` file per locale to the output directory (default `./suprsend/preference_categories`).

```
suprsend category pull [flags]
```

## Examples

```
  # Pull categories to default directory (suprsend/preference_categories/)
  suprsend category pull

  # Pull to a custom directory
  suprsend category pull --dir ./my-categories

  # Pull draft categories
  suprsend category pull --mode draft
```

### Tips

- Pull overwrites the local `categories.json` and `translations/<locale>.json` files without asking, so commit or stash local edits first. `--force` does not affect overwriting: it only skips the output-directory prompt shown when `--dir` is omitted and `./suprsend/preference_categories` doesn't exist. In non-interactive shells pass `--dir` or `--force`; otherwise the command exits 64 (invalid usage) without pulling anything.
- Defaults to the **live** mode. Use `--mode draft` to mirror the pending state instead.

### Options

```
  -d, --dir string    Directory to save category files to (default: ./suprsend/preference_categories)
  -F, --force         Skip directory confirmation prompt, use default path
  -h, --help          help for pull
  -m, --mode string   Version mode: draft or live (default "live")
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

