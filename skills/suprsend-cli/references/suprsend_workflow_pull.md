# suprsend workflow pull

Pull workflows from SuprSend workspace to local

Download workflow definitions from a workspace to local JSON files. Each workflow is saved as `<dir>/<slug>/workflow.json` (default dir `./suprsend/workflows`). Pass a slug as a positional argument or via --slug to pull a single workflow, or omit to pull all.

```
suprsend workflow pull [<slug>] [flags]
```

## Examples

```
  # Pull all workflows to default directory (suprsend/workflows/)
  suprsend workflow pull

  # Pull a single workflow by slug
  suprsend workflow pull welcome

  # Pull to a custom directory using the flag form
  suprsend workflow pull --slug welcome --dir ./my-workflows
```

### Tips

- Pull overwrites `<dir>/<slug>/workflow.json` for every pulled slug without asking, so commit or stash local edits first. `--force` does not affect overwriting: it only skips the output-directory prompt shown when `--dir` is omitted and `./suprsend/workflows` doesn't exist. In non-interactive shells pass `--dir` or `--force`; otherwise the command exits 64 (invalid usage) without pulling anything.
- Defaults to the **live** mode. Use `--mode draft` to mirror the pending state instead.
- Pull also records what it pulled in `<dir>/.suprsend-state.json`, per workspace, so the next push can tell what changed locally and what changed in the workspace since. Commit that file with the workflows. The record is of the workspace's **draft** (one more request per workflow list, since push compares with the draft), so a draft that differs from the live version you pulled isn't mistaken for a later change.

### Options

```
  -d, --dir string    Directory to save workflow files to (default: ./suprsend/workflows)
  -F, --force         Skip directory confirmation prompt, use default path
  -h, --help          help for pull
  -m, --mode string   Version mode: draft or live (default "live")
  -g, --slug string   Workflow slug to pull (omit to pull all)
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

