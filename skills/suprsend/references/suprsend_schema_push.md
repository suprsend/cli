# suprsend schema push

Push schemas

Push schemas in a workspace

```
suprsend schema push [flags]
```

### Options

```
  -c, --commit string           Commit the schemas (--commit=true) (default "true")
  -m, --commit-message string   Commit message describing the changes for --commit=true
  -d, --dir string              Directory for schemas pull to (default: ./suprsend/schema)
  -h, --help                    help for push
  -g, --slug string             Slug of schema to push
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
  -n, --no-color               Disable color output (default: $NO_COLOR)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace to use the schemas from (default "staging")
```

