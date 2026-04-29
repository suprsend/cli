# suprsend template get

Get template details including variants

Retrieve a specific template by slug, including all its channel variants and mock data. Use --mode to switch between draft and live versions.

```
suprsend template get [<slug>] [flags]
```

## Examples

```
  # Get a template by slug (positional)
  suprsend template get welcome-email

  # Get using the flag form
  suprsend template get --slug welcome-email

  # Get the draft version
  suprsend template get welcome-email --mode draft
```

### Tips

- Use `-o json` for machine-readable JSON output, `-o yaml` for YAML. Default `-o json` outputs the full template with variants.

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
  -q, --quiet                  Suppress info/warn output (errors are still shown)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace to list templates from (default "staging")
```

