# suprsend schema list

List schemas

List trigger payload schemas in a workspace with pagination. Returns schema slug, name, and version info. Use --mode to switch between draft and live versions.

```
suprsend schema list [flags]
```

### Tips

- Use `-o json` for machine-readable JSON output, `-o yaml` for YAML. Default `-o pretty` outputs a human-friendly table.

### Options

```
  -h, --help            help for list
  -l, --limit int       Maximum number of schemas to return (default 20)
  -m, --mode string     Version mode: draft or live (default "live")
  -f, --offset int      Number of schemas to skip for pagination
  -o, --output string   Output format: pretty, json, or yaml (default "pretty")
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
  -n, --no-color               Disable color output (default: $NO_COLOR)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace name (e.g., staging, production) (default "staging")
```

