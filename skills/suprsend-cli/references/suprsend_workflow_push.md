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
- `--dry-run` makes no API calls: it only reads and parses the local files (or `--json`) and reports what would be pushed. Nothing is validated server-side.
- With `--commit`, a workflow that is saved as a draft but not committed is an error. Pushing one workflow (`workflow push <slug> --commit`) exits 1 for an invalid workflow (code `api-validation-failed`, with the validation errors in the message) and for any other commit failure. Pushing all workflows exits 1 if any of them fails, and the error lists each failed workflow with its reason. A workflow whose draft already matches live counts as success (nothing to commit).
- `-F/--force` only skips the confirmation prompt for `--commit`; it does not change what is committed. The prompt only appears when stdin is a TTY, so in non-interactive shells (CI, agents) `--commit` proceeds without asking even without `-F`.
- Each workflow is reported on stdout as `created` or `updated` (saved as draft), `unchanged` (the workspace's draft already matches the file, so nothing was sent; with `--commit`, the draft already matched live), `skipped` (see the state tip), `committed` or `failed`, followed by a summary; with `-o json` the result is one JSON object with `items` and `counts`.
- Pull and push keep `.suprsend-state.json` in the workflows directory (commit it with the files): what each workflow looked like, per workspace, at the last pull or push. Push compares the file and the workspace's draft with it: a workflow changed only in the workspace since then is `skipped` rather than overwritten with the stale file (with `--commit` that is an error, so nobody else's draft goes live unseen), and one changed on both sides fails as a conflict (code `api-conflict`). Pull again to get the workspace's version, or pass `--overwrite` to push the file anyway. Without a record for that workspace (a first push, or files pulled from another workspace), push sends every workflow that differs from the draft. A pull takes the live version by default, while push compares with the draft. If the draft differed from live at the pull, it is someone's pending work: push reports the workflow `unchanged` with a note and leaves the draft alone (with `--commit` it is not committed), and a local edit to it fails as a conflict. Pull with `--mode draft` to work on the draft, or pass `--overwrite` to replace it. The order of a multi-channel node's `channels` is ignored when comparing.
- Enabled/disabled is a property of the workflow, not a version: an `is_enabled` in the file applies on push, without `--commit`. `workflow pull` writes the key. If a file has no `is_enabled`, a push leaves the workflow's enabled state as it is; a new workflow starts enabled. Use `workflow enable` / `workflow disable` to change only that.

### Options

```
  -c, --commit                  Promote changes from draft to live after pushing
      --commit-message string   Message describing the changes being committed
  -d, --dir string              Directory containing workflow subdirectories (default: ./suprsend/workflows)
  -n, --dry-run                 Report what would be pushed without making any changes
  -F, --force                   Skip confirmation prompt when --commit is set
  -h, --help                    help for push
  -j, --json string             Workflow definition as a JSON object (requires --slug). Must be a valid workflow object, e.g. '{"name":"My Workflow","nodes":[...]}'
      --overwrite               Push your copy even when the workspace's version changed since the last pull or push, replacing that change
  -g, --slug string             Workflow slug to push (omit to push all)
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
      --no-color               Disable color output (default: $NO_COLOR)
  -o, --output string          Output format: pretty, json, or yaml (default "pretty")
      --profile name           Profile to use, by name (default: $SUPRSEND_PROFILE, else the config file's active profile)
  -q, --quiet                  Suppress info and warning output (results and errors are still shown)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace slug, e.g. staging or production (default: $SUPRSEND_WORKSPACE, else the profile's workspace, else staging)
```

