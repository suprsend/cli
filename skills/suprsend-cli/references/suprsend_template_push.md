# suprsend template push

Push templates and their variants from local to SuprSend workspace

```
suprsend template push [flags]
```

### Options

```
  -c, --commit                  Commit the pushed templates to live
  -m, --commit-message string   Commit message describing the changes
  -d, --dir string              Input directory for templates (default: ./suprsend/templates)
  -f, --force                   Force commit by skipping variants with errors
  -h, --help                    help for push
  -g, --slug string             Slug of a specific template to push
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
  -n, --no-color               Disable color output (default: $NO_COLOR)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace to list templates from (default "staging")
```

