# suprsend workflow pull

Pull workflows from SuprSend workspace to local

Download workflow definitions from a workspace to local JSON files. Saves one JSON file per workflow (named by slug) to the output directory. Pass a slug as a positional argument or via --slug to pull a single workflow, or omit to pull all.

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

- Pull overwrites local workflow JSON files for the matched slugs. Commit local edits first if you don't want them clobbered (or use `--force` to skip the prompt).
- Defaults to the **live** mode. Use `--mode draft` to mirror the pending state instead.

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
  -q, --quiet                  Suppress info/warn output (errors are still shown)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace name (e.g., staging, production) (default "staging")
```

