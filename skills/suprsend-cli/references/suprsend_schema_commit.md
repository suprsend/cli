# suprsend schema commit

Commit schema from draft to live

Promote a schema from draft to live mode. Requires a schema slug as a positional argument. Once committed, the schema changes become active immediately.

```
suprsend schema commit [flags]
```

### Options

```
  -m, --commit-message string   Message describing the changes being committed
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

