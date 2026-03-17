## suprsend schema push

Push schemas

### Synopsis

Upload local schema JSON files to a workspace. Reads .json files from the input directory and pushes them. By default, changes are committed immediately (--commit=true). Use --slug to push a single schema.

```
suprsend schema push [flags]
```

### Options

```
  -c, --commit string           Promote changes from draft to live after pushing (true/false) (default "true")
  -m, --commit-message string   Message describing the changes being committed
  -d, --dir string              Directory containing schema JSON files (default: ./suprsend/schema)
  -h, --help                    help for push
  -g, --slug string             Schema slug to push (omit to push all)
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

* [suprsend schema](suprsend_schema.md)	 - Manage trigger payload schemas

