# suprsend template commit

Commit a template from draft to live

Commit a template from draft to live in a workspace. Pass the template slug as a positional argument or via --slug. Once committed, the template changes become visible to users.

```
suprsend template commit [<slug>] [flags]
```

## Examples

```
  # Commit a template to live (positional slug)
  suprsend template commit welcome-email

  # Commit using the flag form
  suprsend template commit --slug welcome-email

  # Dry run: see what would be committed without making changes
  suprsend template commit welcome-email --dry-run
```

### Tips

- Commit is irreversible: it promotes the draft to **live**, overwriting the previous live version. Affected workflows immediately render the new content.
- If you didn't author the draft locally, run `suprsend template get --slug <slug> --mode draft` first to inspect what will become live.

### Options

```
      --commit-message string   Commit message describing the changes
  -n, --dry-run                 Print what would be committed without making any changes
  -F, --force                   Force commit by skipping variants with errors
  -h, --help                    help for commit
  -g, --slug string             Template slug
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

