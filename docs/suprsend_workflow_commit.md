## suprsend workflow commit

Commit workflow from draft to live

### Synopsis

Promote a workflow from draft to live mode. Requires a workflow slug as a positional argument. Once committed, the workflow changes become active immediately.

```
suprsend workflow commit [flags]
```

### Options

```
  -h, --help                    help for commit
  -m, --commit-message string   Message describing the changes being committed
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
  -n, --no-color               Disable color output (default: $NO_COLOR)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace name (e.g., staging, production) (default "staging")
```

### SEE ALSO

* [suprsend workflow](suprsend_workflow.md)	 - Manage workflows
