## suprsend category translation list

List preference translations

### Synopsis

List available translation locales for preference categories in a workspace. Returns the locale codes that have translations configured.

```
suprsend category translation list [flags]
```

### Examples

```
  # List available translation locales
  suprsend category translation list

  # List with JSON output
  suprsend category translation list --output json

  # List in the production workspace
  suprsend category translation list --workspace production
```

### Options

```
  -h, --help   help for list
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
      --no-color               Disable color output (default: $NO_COLOR)
  -q, --quiet                  Suppress all log output (only fatal errors are shown)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace name (e.g., staging, production) (default "staging")
```

### SEE ALSO

* [suprsend category translation](suprsend_category_translation.md)	 - Manage preference category translations

