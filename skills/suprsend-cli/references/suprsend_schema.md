# suprsend schema

Manage trigger payload schemas

Manage trigger payload schemas. Schemas define the JSON structure for workflow and event trigger payloads. Subcommands let you list, pull, push, and commit schemas.

```
suprsend schema [flags]
```

### Options

```
  -h, --help                   help for schema
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -w, --workspace string       Workspace name (e.g., staging, production) (default "staging")
```

### Options inherited from parent commands

```
      --config string      config file (default: $HOME/.suprsend.yaml)
  -n, --no-color           Disable color output (default: $NO_COLOR)
  -v, --verbosity string   Log level (debug, info, warn, error, fatal, panic) (default "info")
```

