## suprsend workflow

Manage workflows

### Synopsis

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

### SEE ALSO

* [suprsend](suprsend.md)	 - CLI to interact with SuprSend, a Notification Infrastructure
* [suprsend workflow disable](suprsend_workflow_disable.md)	 - Disable a workflow
* [suprsend workflow enable](suprsend_workflow_enable.md)	 - Enables a workflow.
* [suprsend workflow get](suprsend_workflow_get.md)	 - Get workflow details
* [suprsend workflow list](suprsend_workflow_list.md)	 - List workflows for a workspace
* [suprsend workflow pull](suprsend_workflow_pull.md)	 - Pull workflows from SuprSend workspace to local
* [suprsend workflow push](suprsend_workflow_push.md)	 - Push workflows from local to SuprSend workspace

