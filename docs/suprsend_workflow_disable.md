## suprsend workflow disable

Disable a workflow

### Synopsis

Disable a workflow to stop it from processing triggers. Requires a workflow slug as a positional argument.

```
suprsend workflow disable [flags]
```

### Options

```
  -h, --help   help for disable
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

