# suprsend sync

Sync SuprSend assets from one workspace to another

```
suprsend sync [flags]
```

### Options

```
  -a, --assets string   Assets to sync (all, workflow, schema, event, category, translation) (default "all")
  -d, --dir string      Directory to sync assets to
  -f, --from string     Source workspace (required) (default "staging")
  -h, --help            help for sync
  -m, --mode string     Mode to sync assets (draft, live), default: live (default "live")
  -t, --to string       Destination workspace (required) (default "production")
```

### Options inherited from parent commands

```
      --config string      config file (default: $HOME/.suprsend.yaml)
  -n, --no-color           Disable color output (default: $NO_COLOR)
  -v, --verbosity string   Log level (debug, info, warn, error, fatal, panic) (default "info")
```

