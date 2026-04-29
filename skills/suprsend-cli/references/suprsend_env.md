# suprsend env

Show recognized environment variables and their current values

Print all environment variables recognized by suprsend, their current values,
and how each one affects the CLI. Values for sensitive variables (tokens) are
redacted. Useful for verifying configuration in CI/CD pipelines and agent contexts.

```
suprsend env [flags]
```

## Examples

```
  suprsend env
```

### Options

```
  -h, --help   help for env
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
      --no-color               Disable color output (default: $NO_COLOR)
  -o, --output string          Output format: pretty, json, or yaml (default "pretty")
  -q, --quiet                  Suppress info/warn output (errors are still shown)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
```

