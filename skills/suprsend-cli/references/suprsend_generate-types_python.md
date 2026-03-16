# suprsend generate-types python

Generate Python types from JSON Schema

```
suprsend generate-types python [flags]
```

### Options

```
  -h, --help                 help for python
      --mode string          Mode of schema to fetch (draft, live), default: live (default "live")
      --output-file string   Output file for generated Python types (default "suprsend_types.py")
      --pydantic             Generate Pydantic types for Python (default true)
      --workspace string     Workspace to get schemas from. (default "staging")
```

### Options inherited from parent commands

```
      --config string      config file (default: $HOME/.suprsend.yaml)
  -n, --no-color           Disable color output (default: $NO_COLOR)
  -v, --verbosity string   Log level (debug, info, warn, error, fatal, panic) (default "info")
```

