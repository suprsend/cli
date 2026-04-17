## suprsend template get

Get template details including variants

### Synopsis

Retrieve a specific template by slug, including all its channel variants and mock data. Requires --slug. Use --mode to switch between draft and live versions.

```
suprsend template get [flags]
```

### Options

```
  -h, --help            help for get
  -m, --mode string     Version mode: draft or live (default "live")
  -o, --output string   Output format: json, yaml, or pretty (default "json")
  -g, --slug string     Template slug to retrieve (required)
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
  -n, --no-color               Disable color output (default: $NO_COLOR)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace to list templates from (default "staging")
```

### SEE ALSO

* [suprsend template](suprsend_template.md)	 - Manage templates

