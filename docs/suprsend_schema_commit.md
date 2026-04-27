## suprsend schema commit

Commit schema from draft to live

### Synopsis

Promote a schema from draft to live mode. Pass the schema slug as a positional argument or via --slug. Once committed, the schema changes become active immediately.

```
suprsend schema commit [<slug>] [flags]
```

### Examples

```
  # Commit a schema to live (positional slug)
  suprsend schema commit order-placed

  # Commit using the flag form
  suprsend schema commit --slug order-placed

  # Commit in the production workspace
  suprsend schema commit order-placed --workspace production
```

### Options

```
      --commit-message string   Message describing the changes being committed
  -n, --dry-run                 Print what would be committed without making any changes
  -F, --force                   Skip confirmation prompt
  -h, --help                    help for commit
  -g, --slug string             Schema slug
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
      --no-color               Disable color output (default: $NO_COLOR)
  -q, --quiet                  Suppress all log output (only fatal errors are shown)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace name (e.g., staging, production) (default "staging")
```

### SEE ALSO

* [suprsend schema](suprsend_schema.md)	 - Manage trigger payload schemas

