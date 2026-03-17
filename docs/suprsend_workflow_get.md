## suprsend workflow get

Get workflow details

### Synopsis

Retrieve detailed information for a specific workflow by its slug. Requires --slug. Returns the full workflow definition including nodes, connections, and configuration. Use --mode to switch between draft and live versions.

```
suprsend workflow get [flags]
```

### Options

```
  -h, --help            help for get
      --mode string     Version mode: draft or live (default "live")
  -o, --output string   Output format: json or yaml (default "json")
  -g, --slug string     Workflow slug to retrieve (required)
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

