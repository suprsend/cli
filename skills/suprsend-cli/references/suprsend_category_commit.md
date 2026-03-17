# suprsend category commit

Commit categories

Promote preference categories from draft to live mode. Also pushes any local translation files from the translations subdirectory before committing.

```
suprsend category commit [flags]
```

### Options

```
      --commit-message string   Message describing the changes being committed
  -d, --dir string              Directory containing category and translation files (default: ./suprsend/category)
  -h, --help                    help for commit
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
  -n, --no-color               Disable color output (default: $NO_COLOR)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace name (e.g., staging, production) (default "staging")
```

