# suprsend schema commit

Commit schema from draft to live

Commit schema from draft to live in a workspace. Example: suprsend schema commit <slug>

```
suprsend schema commit [flags]
```

### Options

```
  -m, --commit-message string   Commit message describing the changes
  -h, --help                    help for commit
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
  -n, --no-color               Disable color output (default: $NO_COLOR)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace to use the schemas from (default "staging")
```

