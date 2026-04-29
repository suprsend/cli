# suprsend profile remove

Remove a profile

Remove a profile from the configs. Only useful if you have a BYOC/self-hosted SuprSend instance or if you want to manage multiple accounts. Not required for moving assets between workspaces in the same account.

```
suprsend profile remove [flags]
```

### Options

```
  -h, --help          help for remove
      --name string   Name of the profile to remove
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

