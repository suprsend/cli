# suprsend workflow push

Push workflows from local to SuprSend workspace

Upload local workflow JSON files to a workspace. Reads .json files from the input directory and pushes them. By default, changes are staged as drafts. Use --commit to also promote to live. Pass a slug as a positional argument or via --slug to push a single workflow, or omit to push all.

```
suprsend workflow push [<slug>] [flags]
```

## Examples

```
  # Push all workflows from default directory
  suprsend workflow push

  # Push a single workflow and commit to live immediately
  suprsend workflow push welcome --commit

  # Dry run: preview what would be pushed without making changes
  suprsend workflow push --dry-run
```

### Tips

- Push writes to the **draft** state. Run `suprsend workflow commit` to promote draft → live.
- Pair with `--dry-run` to validate the workflow server-side without writing to the draft. Pair with `--commit` to push + commit in one step.

### Options

```
  -c, --commit                  Promote changes from draft to live after pushing
      --commit-message string   Message describing the changes being committed
  -d, --dir string              Directory containing workflow subdirectories (default: ./suprsend/workflows)
  -n, --dry-run                 Print what would be pushed without making any changes
  -h, --help                    help for push
  -j, --json string             Workflow definition as a JSON object (requires --slug). Must be a valid workflow object, e.g. '{"name":"My Workflow","nodes":[...]}'
  -g, --slug string             Workflow slug to push (omit to push all)
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
      --no-color               Disable color output (default: $NO_COLOR)
  -o, --output string          Output format: pretty, json, or yaml (default "pretty")
  -q, --quiet                  Suppress info/warn output (errors are still shown)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace name (e.g., staging, production) (default "staging")
```

