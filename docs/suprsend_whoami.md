## suprsend whoami

Show which profile, host and workspace commands will use

### Synopsis

Print where commands will go: the active profile, the management API host,
the workspace a command would use (and where that value came from: -w,
SUPRSEND_WORKSPACE, the profile, or the built-in default), and where the
service token came from. The token itself is never printed. With --check, it
also calls the API to confirm the token works and the workspace exists.

```
suprsend whoami [flags]
```

### Examples

```
  # Show the target of commands run from this shell
  suprsend whoami

  # Show the target for a specific workspace and confirm it exists
  suprsend whoami -w production --check

  # Machine-readable
  suprsend whoami -o json
```

### Options

```
      --check              Also call the API to confirm the token works and the workspace exists
  -h, --help               help for whoami
  -w, --workspace string   Workspace slug, e.g. staging or production (default: $SUPRSEND_WORKSPACE, else the profile's workspace, else staging)
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

### SEE ALSO

* [suprsend](suprsend.md)	 - CLI to interact with SuprSend, a Notification Infrastructure

