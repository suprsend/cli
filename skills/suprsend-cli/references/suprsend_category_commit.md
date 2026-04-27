# suprsend category commit

Commit categories

Promote preference categories from draft to live mode. Also pushes any local translation files from the translations subdirectory before committing.

```
suprsend category commit [flags]
```

## Examples

```
  # Commit categories to live
  suprsend category commit

  # Commit in the production workspace
  suprsend category commit --workspace production

  # Commit with a message
  suprsend category commit --commit-message "Update notification preferences"
```

### Options

```
      --commit-message string   Message describing the changes being committed
  -d, --dir string              Directory containing category and translation files (default: ./suprsend/preference_categories)
  -n, --dry-run                 Print what would be committed without making any changes
  -F, --force                   Skip confirmation prompt
  -h, --help                    help for commit
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
      --no-color               Disable color output (default: $NO_COLOR)
  -q, --quiet                  Suppress all log output (only fatal errors are shown)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace name (e.g., staging, production) (default "staging")
```

