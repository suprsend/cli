# suprsend schema push

Push schemas

Upload local schema JSON files to a workspace. Reads .json files from the input directory and pushes them. By default, changes are staged as drafts. Use --commit to also promote to live. Pass a slug as a positional argument or via --slug to push a single schema.

```
suprsend schema push [<slug>] [flags]
```

## Examples

```
  # Push all schemas from default directory
  suprsend schema push

  # Push a single schema and commit to live immediately
  suprsend schema push order-placed --commit

  # Dry run: preview what would be pushed without making changes
  suprsend schema push --dry-run
```

### Options

```
  -c, --commit                  Promote changes from draft to live after pushing
      --commit-message string   Message describing the changes being committed
  -d, --dir string              Directory containing schema files (default: ./suprsend/schemas)
  -n, --dry-run                 Print what would be pushed without making any changes
  -h, --help                    help for push
  -j, --json string             Schema definition as a JSON object (requires --slug). Must be a valid JSON Schema object, e.g. '{"type":"object","properties":{"key":{"type":"string"}}}'
  -g, --slug string             Schema slug to push (omit to push all)
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

