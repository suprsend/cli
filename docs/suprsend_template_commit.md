## suprsend template commit

Commit a template from draft to live

### Synopsis

Commit a template from draft to live in a workspace. Pass the template slug as a positional argument or via --slug. Once committed, the template changes become visible to users.

```
suprsend template commit [<slug>] [flags]
```

### Examples

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

### SEE ALSO

* [suprsend template](suprsend_template.md)	 - Manage templates

