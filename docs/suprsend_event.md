## suprsend event

Manage events

### Synopsis

Manage events. Subcommands let you list events, pull event definitions to local files, and push event-schema mappings to a workspace.

```
suprsend event [flags]
```

### Options

```
  -h, --help                   help for event
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
* [suprsend event list](suprsend_event_list.md)	 - List events
* [suprsend event pull](suprsend_event_pull.md)	 - Pull events from workspace to local directory
* [suprsend event push](suprsend_event_push.md)	 - Push linked events

