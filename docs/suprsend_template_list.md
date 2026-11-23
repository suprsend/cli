## suprsend template list

List templates for a workspace

### Synopsis

List the templates in a workspace: all of them unless --limit is given. Returns template slug, name, and enabled channel info. Use --mode to switch between draft and live versions.

```
suprsend template list [flags]
```

### Examples

```
  # List all templates (live mode)
  suprsend template list

  # List draft templates
  suprsend template list --mode draft

  # Paginate with JSON output
  suprsend template list --limit 50 --output json
```

### Options

```
  -h, --help            help for list
  -l, --limit int       Maximum number of templates to return (default: all)
  -m, --mode string     Version mode: draft or live (default "live")
      --offset int      Offset the number of templates to list (default: 0)
  -o, --output string   Output Style (pretty, yaml, json) (default "pretty")
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
      --no-color               Disable color output (default: $NO_COLOR)
      --profile name           Profile to use, by name (default: $SUPRSEND_PROFILE, else the config file's active profile)
  -q, --quiet                  Suppress info and warning output (results and errors are still shown)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace slug, e.g. staging or production (default: $SUPRSEND_WORKSPACE, else the profile's workspace, else staging)
```

### SEE ALSO

* [suprsend template](suprsend_template.md)	 - Manage templates

