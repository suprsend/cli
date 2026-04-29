## suprsend translation get

Get translations

### Synopsis

Retrieve all template translations from a workspace. Returns translation content keyed by locale code. Use --mode to switch between draft and live versions.

```
suprsend translation get [flags]
```

### Examples

```
  # Get all translations (live mode)
  suprsend translation get

  # Get draft translations
  suprsend translation get --mode draft

  # Get with JSON output
  suprsend translation get --output json
```

### Options

```
  -h, --help            help for get
  -m, --mode string     Version mode: draft or live (default "live")
  -o, --output string   Output format: json or yaml (default "json")
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

### SEE ALSO

* [suprsend translation](suprsend_translation.md)	 - Manage Translations

