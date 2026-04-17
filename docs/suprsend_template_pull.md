## suprsend template pull

Pull templates and their variants from SuprSend workspace

### Synopsis

Pull templates and their variants from SuprSend workspace

```
suprsend template pull [flags]
```

### Options

```
  -d, --dir string    Output directory for templates (default: ./suprsend/templates)
  -f, --force         Force using default directory without prompting
  -h, --help          help for pull
  -m, --mode string   Mode of templates to pull (draft, live) (default "live")
  -g, --slug string   Slug of a specific template to pull
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

