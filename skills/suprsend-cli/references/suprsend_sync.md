# suprsend sync

Sync SuprSend assets from one workspace to another

Sync notification assets from one workspace to another. Pulls assets from the source workspace and pushes them to the destination. Supports syncing all asset types or a specific type (workflow, schema, event, category, translation). Source and destination workspaces must be different.

```
suprsend sync [flags]
```

### Options

```
  -a, --assets string   Asset types to sync: all, workflow, schema, event, category, or translation (default "all")
  -d, --dir string      Local directory for intermediate file storage during sync
  -f, --from string     Source workspace to pull assets from (default "staging")
  -h, --help            help for sync
  -m, --mode string     Version mode: draft or live (default "live")
  -t, --to string       Destination workspace to push assets to (default "production")
```

### Options inherited from parent commands

```
      --config string      config file (default: $HOME/.suprsend.yaml)
  -n, --no-color           Disable color output (default: $NO_COLOR)
  -v, --verbosity string   Log level (debug, info, warn, error, fatal, panic) (default "info")
```

