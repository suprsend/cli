# suprsend workflow push

Push workflows from local to SuprSend workspace

Upload local workflow definitions to a workspace. Reads `<dir>/<slug>/workflow.json` for every subdirectory of the input directory (default `./suprsend/workflows`); the directory name is used as the slug. By default, changes are staged as drafts. Use --commit to also promote to live. Pass a slug as a positional argument or via --slug to push a single workflow, or omit to push all. With --slug, --json pushes an inline workflow definition instead of reading a file.

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

- Push writes to the **draft** state. Run `suprsend workflow commit` to promote draft → live, or pass `--commit` to push + commit in one step.
- `--dry-run` makes no API calls: it only reads and parses the local files (or `--json`) and logs what would be pushed. Nothing is validated server-side.
- With `--commit`, a workflow that is saved as a draft but not committed is an error. Pushing one workflow (`workflow push <slug> --commit`) exits 2 for an invalid workflow, with the validation errors in the message, and 1 for any other commit failure. Pushing all workflows exits 1 if any of them fails; the per-workflow reasons are only in the log output. A workflow whose draft already matches live counts as success (nothing to commit).
- `-F/--force` only skips the confirmation prompt for `--commit`; it does not change what is committed. The prompt only appears when stdin is a TTY, so in non-interactive shells (CI, agents) `--commit` proceeds without asking even without `-F`.

### Options

```
  -c, --commit                  Promote changes from draft to live after pushing
      --commit-message string   Message describing the changes being committed
  -d, --dir string              Directory containing workflow subdirectories (default: ./suprsend/workflows)
  -n, --dry-run                 Print what would be pushed without making any changes
  -F, --force                   Skip confirmation prompt when --commit is set
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

