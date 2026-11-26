# suprsend whoami

Show which profile, host and workspace commands will use

Print where commands will go: the profile (and where it came from:
--profile, SUPRSEND_PROFILE, or the config file's active profile), the management API host,
the workspace a command would use (and where that value came from: -w,
SUPRSEND_WORKSPACE, the profile, or the built-in default), and where the
service token came from. The token itself is never printed. With --check, it
also calls the API to confirm the token works and the workspace exists. When
no workspace is set anywhere, --check confirms the token only: it does not
check the built-in default, so pass -w to check the workspace you mean.

```
suprsend whoami [flags]
```

## Examples

```
  # Show the target of commands run from this shell
  suprsend whoami

  # Show the target for a specific workspace and confirm it exists
  suprsend whoami -w production --check

  # Machine-readable
  suprsend whoami -o json
```

### Tips

- Run `suprsend whoami --check -w <workspace> -o json` before writing to a workspace: it shows the host, profile and workspace commands will use without reading the config file (which holds tokens), and fails if the token is invalid or the workspace doesn't exist. Always pass `-w` (or have `SUPRSEND_WORKSPACE` set): with no workspace set, only the token is checked (`workspace_checked` is false) and `workspace_source` is `default`.

### Options

```
      --check              Also call the API to confirm the token works and the workspace exists (the token only, when no workspace is set)
  -h, --help               help for whoami
  -w, --workspace string   Workspace slug, e.g. staging or production (default: $SUPRSEND_WORKSPACE, else the profile's workspace, else staging)
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
      --no-color               Disable color output (default: $NO_COLOR)
  -o, --output string          Output format: pretty, json, or yaml (default "pretty")
      --profile name           Profile to use, by name (default: $SUPRSEND_PROFILE, else the config file's active profile)
  -q, --quiet                  Suppress info and warning output (results and errors are still shown)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
```

