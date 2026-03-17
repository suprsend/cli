# suprsend category pull

Pull categories from a workspace

Download preference categories and their translations from a workspace to local files. Saves categories_preferences.json and locale-specific translation files to the output directory.

```
suprsend category pull [flags]
```

### Options

```
  -d, --dir string    Directory to save category files to (default: ./suprsend/category)
  -f, --force         Skip directory confirmation prompt, use default path
  -h, --help          help for pull
  -m, --mode string   Version mode: draft or live (default "live")
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
  -n, --no-color               Disable color output (default: $NO_COLOR)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace name (e.g., staging, production) (default "staging")
```

