# suprsend workflow pull

Pull workflows from SuprSend workspace to local

```
suprsend workflow pull [flags]
```

### Options

```
  -d, --dir string    Output directory for workflows (default: ./suprsend/workflow)
  -f, --force         Force using default directory without prompting
  -h, --help          help for pull
  -m, --mode string   Mode of workflows to pull from (draft, live) (default "live")
  -g, --slug string   Slug of the workflow to pull
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
  -n, --no-color               Disable color output (default: $NO_COLOR)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace to list workflows from (default "staging")
```

