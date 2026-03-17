# suprsend schema push

Push schemas

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
  -j, --json string             Schema definition as a JSON object (requires --slug). Must be a valid JSON Schema object, e.g. '{"type":"object","properties":{"key":{"type":"string"}}}'
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

