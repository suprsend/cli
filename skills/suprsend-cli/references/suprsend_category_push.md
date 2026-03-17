# suprsend category push

Push categories to a workspace

Upload local preference categories and translations to a workspace. Reads categories_preferences.json and translation files from the input directory. By default, changes are committed immediately (--commit=true).

```
suprsend category push [flags]
```

### Options

```
  -c, --commit string           Promote changes from draft to live after pushing (true/false) (default "true")
  -m, --commit-message string   Message describing the changes being committed
  -d, --dir string              Directory containing category files (default: ./suprsend/category/)
  -h, --help                    help for push
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
  -n, --no-color               Disable color output (default: $NO_COLOR)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace name (e.g., staging, production) (default "staging")
```

