# suprsend category translation push

Push preference translations

Upload local preference category translation files to a workspace. Reads {locale}.json files from the input directory. English translations cannot be pushed. Use --locale to push a single locale, or omit to push all.

```
suprsend category translation push [flags]
```

### Options

```
  -d, --dir string      Directory containing translation JSON files (default: suprsend/category/translation)
  -h, --help            help for push
  -l, --locale string   Locale code to push, e.g., es, fr (omit to push all)
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
  -n, --no-color               Disable color output (default: $NO_COLOR)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace name (e.g., staging, production) (default "staging")
```

