# suprsend workflow

Manage workflows

Manage workflows. Subcommands let you list, get details, pull to local files, push from local files, and enable/disable workflows in a workspace.

```
suprsend workflow [flags]
```

### Options

```
  -h, --help                   help for workflow
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -w, --workspace string       Workspace name (e.g., staging, production) (default "staging")
```

### Options inherited from parent commands

```
      --config string      config file (default: $HOME/.suprsend.yaml)
  -n, --no-color           Disable color output (default: $NO_COLOR)
  -v, --verbosity string   Log level (debug, info, warn, error, fatal, panic) (default "info")
```

