# suprsend translation push

Push translation files to a workspace

Upload local template translation JSON files to a workspace. Reads all .json files from the input directory and pushes them. Use --commit=true to also finalize the changes immediately.

```
suprsend translation push [flags]
```

### Options

```
  -c, --commit string           Promote changes from draft to live after pushing (true/false) (default "false")
  -m, --commit-message string   Message describing the changes being committed
  -d, --dir string              Directory containing translation JSON files (default: ./suprsend/translation)
  -h, --help                    help for push
  -j, --json string             Translations as a JSON object mapping locale codes (without .json extension) to their translation content objects, e.g. '{"en":{"key":"value"},"fr":{"key":"valeur"}}'
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
  -n, --no-color               Disable color output (default: $NO_COLOR)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace name (e.g., staging, production) (default "staging")
```

