# suprsend generate-types go

Generate Go types from JSON Schema

Generate Go type definitions from trigger payload schemas. Produces struct definitions in a single output file with the specified package name.

```
suprsend generate-types go [flags]
```

### Options

```
  -h, --help                 help for go
      --mode string          Version mode: draft or live (default "live")
      --output-file string   Output file path for generated types (default "suprsend_types.go")
      --package string       Go package name for generated structs (default "suprsend")
      --workspace string     Workspace name (e.g., staging, production) (default "staging")
```

### Options inherited from parent commands

```
      --config string      config file (default: $HOME/.suprsend.yaml)
  -n, --no-color           Disable color output (default: $NO_COLOR)
  -v, --verbosity string   Log level (debug, info, warn, error, fatal, panic) (default "info")
```

