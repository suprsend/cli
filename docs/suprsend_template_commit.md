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

  # Dry run: see what would be committed without making changes
  suprsend template commit welcome-email --dry-run
```

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
  -q, --quiet                  Suppress all log output (only fatal errors are shown)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace to list templates from (default "staging")
```

### SEE ALSO

* [suprsend template](suprsend_template.md)	 - Manage templates

