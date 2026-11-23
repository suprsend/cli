## suprsend version

Print the CLI version

```
suprsend version [flags]
```

### Examples

```
  suprsend version
  suprsend version -o json
  suprsend version -o yaml
  suprsend version -o short
```

### Options

```
  -h, --help            help for version
  -o, --output string   Output format. One of: json | pretty | short | yaml (default "pretty")
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
      --no-color               Disable color output (default: $NO_COLOR)
      --profile name           Profile to use, by name (default: $SUPRSEND_PROFILE, else the config file's active profile)
  -q, --quiet                  Suppress info and warning output (results and errors are still shown)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
```

### SEE ALSO

* [suprsend](suprsend.md)	 - CLI to interact with SuprSend, a Notification Infrastructure

