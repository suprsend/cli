# suprsend template push

Push templates and their variants from local to SuprSend workspace

Push templates and their variants from local to SuprSend workspace. Pass a slug as a positional argument or via --slug to push a single template, or omit to push all.

```
suprsend template push [<slug>] [flags]
```

## Examples

```
  # Push all templates from default directory
  suprsend template push

  # Push a single template and commit to live immediately
  suprsend template push welcome-email --commit

  # Dry run: preview what would be pushed without making changes
  suprsend template push --dry-run
```

### Tips

- Push writes to the **draft** state. Run `suprsend template commit` to promote draft → live.
- Pair with `--dry-run` to validate the template server-side without writing to the draft. Pair with `--commit` to push + commit in one step.

### Options

```
  -c, --commit                  Commit the pushed templates to live
      --commit-message string   Commit message describing the changes
  -d, --dir string              Input directory for templates (default: ./suprsend/templates)
  -n, --dry-run                 Print what would be pushed without making any changes
  -F, --force                   Force commit by skipping variants with errors
  -h, --help                    help for push
  -g, --slug string             Slug of a specific template to push
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

