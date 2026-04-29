# suprsend translation get

Get a single translation file

Retrieve a single template translation from a workspace by filename. Pass the filename as a positional argument or via --slug. Returns the translation's content (the localized string map). Use --mode to switch between draft and live versions.

```
suprsend translation get [<filename>] [flags]
```

## Examples

```
  # Get a translation by filename (positional)
  suprsend translation get candidate.es.json

  # Get using the flag form
  suprsend translation get --slug candidate.es.json

  # Get from draft mode
  suprsend translation get candidate.es.json --mode draft

  # Get with YAML output
  suprsend translation get candidate.es.json --output yaml
```

### Tips

- Use `-o json` for machine-readable JSON output, `-o yaml` for YAML.

### Options

```
  -h, --help            help for get
  -m, --mode string     Version mode: draft or live (default "live")
  -o, --output string   Output format: json or yaml (default "json")
  -g, --slug string     Translation filename, e.g. candidate.es.json
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
      --no-color               Disable color output (default: $NO_COLOR)
  -q, --quiet                  Suppress info/warn output (errors are still shown)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace name (e.g., staging, production) (default "staging")
```

