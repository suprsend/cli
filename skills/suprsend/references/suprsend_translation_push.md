# suprsend translation push

push workflows from local to suprsend

```
suprsend translation push [flags]
```

### Options

```
  -c, --commit string           Commit the translation (--commit=true) (default "false")
  -m, --commit-message string   Commit message for the translation
  -d, --dir string              Directory for translations pull to (default: ./suprsend/translation)
  -h, --help                    help for push
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
  -n, --no-color               Disable color output (default: $NO_COLOR)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace to list translations for (default "staging")
```

