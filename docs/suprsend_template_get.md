## suprsend template get

Get template details including variants

### Synopsis

Retrieve a specific template by slug, including all its channel variants and mock data. Use --mode to switch between draft and live versions.

```
suprsend template get [<slug>] [flags]
```

### Examples

```
  # Get a template by slug (positional)
  suprsend template get welcome-email

  # Get using the flag form
  suprsend template get --slug welcome-email

  # Get the draft version
  suprsend template get welcome-email --mode draft
```

### Options

```
  -h, --help            help for get
  -m, --mode string     Version mode: draft or live (default "live")
  -o, --output string   Output format: json, yaml, or pretty (default "json")
  -g, --slug string     Template slug
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

