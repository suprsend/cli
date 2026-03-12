# suprsend generate-types go

Generate Go types from JSON Schema

```
suprsend generate-types go [flags]
```

### Options

```
  -h, --help                 help for go
      --mode string          Mode of schema to fetch (draft, live), default: live (default "live")
      --output-file string   Output file for generated Go types (default "suprsend_types.go")
      --package string       Package name for Go types (default "suprsend")
      --workspace string     Workspace to get schemas from. (default "staging")
```

### Options inherited from parent commands

```
      --config string      config file (default: $HOME/.suprsend.yaml)
  -n, --no-color           Disable color output (default: $NO_COLOR)
  -v, --verbosity string   Log level (debug, info, warn, error, fatal, panic) (default "info")
```

