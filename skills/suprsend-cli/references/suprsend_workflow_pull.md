# suprsend workflow pull

Pull workflows from SuprSend workspace to local

Download workflow definitions from a workspace to local JSON files. Saves one JSON file per workflow (named by slug) to the output directory. Use --slug to pull a single workflow, or omit to pull all.

```
suprsend workflow pull [flags]
```

### Options

```
  -d, --dir string    Directory to save workflow files to (default: ./suprsend/workflow)
  -f, --force         Skip directory confirmation prompt, use default path
  -h, --help          help for pull
  -m, --mode string   Version mode: draft or live (default "live")
  -g, --slug string   Workflow slug to pull (omit to pull all)
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
  -n, --no-color               Disable color output (default: $NO_COLOR)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace name (e.g., staging, production) (default "staging")
```

