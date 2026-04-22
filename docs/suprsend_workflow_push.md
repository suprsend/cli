## suprsend workflow push

Push workflows from local to SuprSend workspace

### Synopsis

Upload local workflow JSON files to a workspace. Reads .json files from the input directory and pushes them. By default, changes are staged as drafts. Use --commit to also promote to live. Use --slug to push a single workflow, or omit to push all.

```
suprsend workflow push [flags]
```

### Options

```
  -c, --commit                  Promote changes from draft to live after pushing
  -m, --commit-message string   Message describing the changes being committed (required when --commit is set)
  -d, --dir string              Directory containing workflow JSON files (default: ./suprsend/workflows)
  -h, --help                    help for push
  -j, --json string             Workflow definition as a JSON object (requires --slug). Must be a valid workflow object, e.g. '{"name":"My Workflow","nodes":[...]}'
  -g, --slug string             Workflow slug to push (omit to push all)
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

