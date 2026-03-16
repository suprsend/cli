# suprsend schema pull

Pull schemas

Pull schemas in a workspace

```
suprsend schema pull [flags]
```

### Options

```
  -d, --dir string    Directory to pull schemas (default: ./suprsend/schema)
  -f, --force         Force using default directory without prompting
  -h, --help          help for pull
  -m, --mode string   Mode of schemas to pull (draft, live), default: live (default "live")
  -g, --slug string   Slug of schema to pull
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
  -n, --no-color               Disable color output (default: $NO_COLOR)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace to use the schemas from (default "staging")
```

