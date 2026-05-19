# suprsend schema pull

Pull schemas

Download schema definitions from a workspace to local JSON files. Saves one JSON file per schema (named by slug) to the output directory. Pass a slug as a positional argument or via --slug to pull a single schema, or omit to pull all.

```
suprsend schema pull [<slug>] [flags]
```

## Examples

```
  # Pull all schemas to default directory (suprsend/schemas/)
  suprsend schema pull

  # Pull a single schema by slug
  suprsend schema pull order-placed

  # Pull to a custom directory using the flag form
  suprsend schema pull --slug order-placed --dir ./my-schemas
```

### Tips

- Pull overwrites local schema JSON files for the matched slugs. Commit local edits first if you don't want them clobbered (or use `--force` to skip the prompt).
- Defaults to the **live** mode. Use `--mode draft` to mirror the pending state instead.

### Options

```
  -d, --dir string    Directory to save schema files to (default: ./suprsend/schemas)
  -F, --force         Skip directory confirmation prompt, use default path
  -h, --help          help for pull
  -m, --mode string   Version mode: draft or live (default "live")
  -g, --slug string   Schema slug to pull (omit to pull all)
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

