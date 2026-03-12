# suprsend profile modify

Modify a profile

Modify a profile in the configs. Only useful if you have a BYOC/self-hosted SuprSend instance or if you want to manage multiple accounts. Not required for moving assets between workspaces in the same account.

```
suprsend profile modify [flags]
```

### Options

```
      --base-url string        Base URL (default: https://hub.suprsend.com/)
  -h, --help                   help for modify
      --mgmnt-url string       Management URL (default: https://management-api.suprsend.com/)
      --name string            Name of the profile to modify
      --service-token string   Service Token
```

### Options inherited from parent commands

```
      --config string      config file (default: $HOME/.suprsend.yaml)
  -n, --no-color           Disable color output (default: $NO_COLOR)
  -v, --verbosity string   Log level (debug, info, warn, error, fatal, panic) (default "info")
```

