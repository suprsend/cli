# suprsend schema get

Get schema details

Retrieve the full definition of a specific schema by its slug. Returns the JSON Schema object including type, properties, and validation rules. Requires --slug. Use --mode to switch between draft and live versions.

```
suprsend schema get [flags]
```

### Tips

- Use `-o json` for machine-readable JSON output, `-o yaml` for YAML.

### Options

```
  -h, --help            help for get
      --mode string     Version mode: draft or live (default "live")
  -o, --output string   Output format: json or yaml (default "json")
  -g, --slug string     Schema slug to retrieve (required)
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
  -n, --no-color               Disable color output (default: $NO_COLOR)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace name (e.g., staging, production) (default "staging")
```

