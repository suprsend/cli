# suprsend profile add

Add a new profile

Add a new profile to the configs. Only useful if you have a BYOC/self-hosted SuprSend instance or if you want to manage multiple accounts. Not required for moving assets between workspaces in the same account.

```
suprsend profile add [flags]
```

### Options

```
      --base-url string        Base URL (default: https://hub.suprsend.com/)
  -h, --help                   help for add
      --mgmnt-url string       Management URL (default: https://management-api.suprsend.com/)
      --name string            Name of the profile (required)
      --service-token string   Service token (required)
```

### Options inherited from parent commands

```
      --config string      config file (default: $HOME/.suprsend.yaml)
  -n, --no-color           Disable color output (default: $NO_COLOR)
  -v, --verbosity string   Log level (debug, info, warn, error, fatal, panic) (default "info")
```

