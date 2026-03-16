# suprsend generate-types kotlin

Generate Kotlin types from JSON Schema

```
suprsend generate-types kotlin [flags]
```

### Options

```
  -h, --help                 help for kotlin
      --mode string          Mode of schema to fetch (draft, live), default: live (default "live")
      --output-file string   Output file for generated Kotlin types (default "SuprsendTypes.kt")
      --package string       Package name for Kotlin types (default "suprsend")
      --workspace string     Workspace to get schemas from. (default "staging")
```

### Options inherited from parent commands

```
      --config string      config file (default: $HOME/.suprsend.yaml)
  -n, --no-color           Disable color output (default: $NO_COLOR)
  -v, --verbosity string   Log level (debug, info, warn, error, fatal, panic) (default "info")
```

