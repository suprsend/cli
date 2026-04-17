# suprsend template commit

Commit a template from draft to live

Commit a template from draft to live in a workspace. Example: suprsend template commit <slug>

```
suprsend template commit <slug> [flags]
```

### Options

```
  -m, --commit-message string   Commit message describing the changes
  -f, --force                   Force commit by skipping variants with errors
  -h, --help                    help for commit
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
  -n, --no-color               Disable color output (default: $NO_COLOR)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace to list templates from (default "staging")
```

