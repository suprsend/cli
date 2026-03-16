# suprsend profile use

Set the active profile

Set the active profile in the configs. Only useful if you have a BYOC/self-hosted SuprSend instance or if you want to manage multiple accounts. Not required for moving assets between workspaces in the same account.

```
suprsend profile use [flags]
```

### Options

```
  -h, --help          help for use
      --name string   Profile name to set as active
```

### Options inherited from parent commands

```
      --config string      config file (default: $HOME/.suprsend.yaml)
  -n, --no-color           Disable color output (default: $NO_COLOR)
  -v, --verbosity string   Log level (debug, info, warn, error, fatal, panic) (default "info")
```

