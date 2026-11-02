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

  # Commit only the variants that pass validation
  suprsend template commit welcome-email --skip-invalid

  # Dry run: see what would be committed without making changes
  suprsend template commit welcome-email --dry-run
```

### Tips

- Commit is irreversible: it promotes the draft to **live**, overwriting the previous live version. Affected workflows immediately render the new content.
- If you didn't author the draft locally, run `suprsend template get --slug <slug> --mode draft` first to inspect what will become live.
- All changed variants are committed together, and one invalid variant fails the commit. `--skip-invalid` runs a pre-commit validation and commits only the variants that pass; the skipped ones are listed in the result (the command fails if every changed variant fails).
- The confirmation prompt only appears when stdin is a TTY. In non-interactive shells (CI, agents) the commit proceeds without asking; `-F/--force` skips it on a TTY too. Use `--dry-run` if you only want to check the command.
- The result says `committed`, or `unchanged` when the draft already matched live. With `-o json` it is one JSON object with `items` and `counts`.

### Options

```
      --commit-message string   Commit message describing the changes
  -n, --dry-run                 Report what would be committed without making any changes
  -F, --force                   Skip the confirmation prompt
  -h, --help                    help for commit
      --skip-invalid            Commit only the variants that pass pre-commit validation and list the ones skipped
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
  -w, --workspace string       Workspace slug, e.g. staging or production (default: $SUPRSEND_WORKSPACE, else the profile's workspace, else staging)
```

