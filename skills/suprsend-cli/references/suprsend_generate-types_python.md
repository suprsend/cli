# suprsend generate-types python

Generate Python types from JSON Schema

Generate Python type definitions from trigger payload schemas. Fetches linked schemas from the workspace and generates a single Python file with type classes. Supports Pydantic models (enabled by default).

```
suprsend generate-types python [flags]
```

### Options

```
  -h, --help                 help for python
      --mode string          Version mode: draft or live (default "live")
      --output-file string   Output file path for generated types (default "suprsend_types.py")
      --pydantic             Generate Pydantic BaseModel classes instead of plain dataclasses (default true)
      --workspace string     Workspace name (e.g., staging, production) (default "staging")
```

### Options inherited from parent commands

```
      --config string      config file (default: $HOME/.suprsend.yaml)
  -n, --no-color           Disable color output (default: $NO_COLOR)
  -v, --verbosity string   Log level (debug, info, warn, error, fatal, panic) (default "info")
```

