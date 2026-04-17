## suprsend template list

List templates for a workspace

### Synopsis

List templates for a workspace

```
suprsend template list [flags]
```

### Options

```
  -h, --help            help for list
  -l, --limit int       Limit the number of templates to list (default 20)
  -m, --mode string     Mode of templates to list (draft, live), default: live (default "live")
  -f, --offset int      Offset the number of templates to list (default: 0)
  -o, --output string   Output Style (pretty, yaml, json) (default "pretty")
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
  -n, --no-color               Disable color output (default: $NO_COLOR)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace to list templates from (default "staging")
```

### SEE ALSO

* [suprsend template](suprsend_template.md)	 - Manage templates

