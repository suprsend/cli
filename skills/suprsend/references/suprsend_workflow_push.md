# suprsend workflow push

Push workflows from local to SuprSend workspace

```
suprsend workflow push [flags]
```

### Options

```
  -c, --commit string           Commit the workflows (--commit=true) (default "true")
  -m, --commit-message string   Commit message describing the changes for --commit=true
  -d, --dir string              Output directory for workflows (default: ./suprsend/workflow)
  -h, --help                    help for push
  -g, --slug string             Slug of the workflow to push
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
  -n, --no-color               Disable color output (default: $NO_COLOR)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace to list workflows from (default "staging")
```

