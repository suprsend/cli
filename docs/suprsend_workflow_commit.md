## suprsend workflow commit

Commit workflow from draft to live

### Synopsis

Promote a workflow from draft to live mode. Pass the workflow slug as a positional argument or via --slug. Once committed, the workflow changes become active immediately.

```
suprsend workflow commit [<slug>] [flags]
```

### Examples

```
  # Commit a workflow to live (positional slug)
  suprsend workflow commit welcome

  # Commit using the flag form
  suprsend workflow commit --slug welcome

  # Dry run: see what would be committed without making changes
  suprsend workflow commit welcome --dry-run
```

### Options

```
      --commit-message string   Message describing the changes being committed
  -n, --dry-run                 Print what would be committed without making any changes
  -F, --force                   Skip confirmation prompt
  -h, --help                    help for commit
  -g, --slug string             Workflow slug
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

* [suprsend workflow](suprsend_workflow.md)	 - Manage workflows

