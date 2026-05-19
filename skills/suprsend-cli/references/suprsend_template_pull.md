# suprsend template pull

Pull templates and their variants from SuprSend workspace

Pull templates and their variants from SuprSend workspace. Pass a slug as a positional argument or via --slug to pull a single template, or omit to pull all.

```
suprsend template pull [<slug>] [flags]
```

## Examples

```
  # Pull all templates to default directory (suprsend/templates/)
  suprsend template pull

  # Pull a single template by slug
  suprsend template pull welcome-email

  # Pull to a custom directory using the flag form
  suprsend template pull --slug welcome-email --dir ./my-templates
```

### Tips

- Pull overwrites local template files for the matched slugs. Commit local edits first if you don't want them clobbered (or use `--force` to skip the prompt).
- Defaults to the **live** mode. Use `--mode draft` to mirror the pending state instead.

### Options

```
  -d, --dir string    Output directory for templates (default: ./suprsend/templates)
  -F, --force         Force using default directory without prompting
  -h, --help          help for pull
  -m, --mode string   Version mode: draft or live (default "live")
  -g, --slug string   Slug of a specific template to pull
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
      --no-color               Disable color output (default: $NO_COLOR)
  -o, --output string          Output format: pretty, json, or yaml (default "pretty")
  -q, --quiet                  Suppress info/warn output (errors are still shown)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace to list templates from (default "staging")
```

