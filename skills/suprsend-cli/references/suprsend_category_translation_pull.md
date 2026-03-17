# suprsend category translation pull

Pull preference translations

Download preference category translations from a workspace to local JSON files. Creates one file per locale (e.g., es.json, fr.json) in the output directory.

```
suprsend category translation pull [flags]
```

### Options

```
  -d, --dir string   Directory to save translation files to (default: suprsend/category/translation)
  -f, --force        Skip directory confirmation prompt, use default path
  -h, --help         help for pull
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
  -n, --no-color               Disable color output (default: $NO_COLOR)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace name (e.g., staging, production) (default "staging")
```

