# suprsend generate-types kotlin

Generate Kotlin types from JSON Schema

Generate Kotlin type definitions from trigger payload schemas. Produces data classes in a single output file with the specified package name.

```
suprsend generate-types kotlin [flags]
```

### Options

```
  -h, --help                 help for kotlin
      --mode string          Version mode: draft or live (default "live")
      --output-file string   Output file path for generated types (default "SuprsendTypes.kt")
      --package string       Kotlin package name for generated data classes (default "suprsend")
      --workspace string     Workspace name (e.g., staging, production) (default "staging")
```

### Options inherited from parent commands

```
      --config string      config file (default: $HOME/.suprsend.yaml)
  -n, --no-color           Disable color output (default: $NO_COLOR)
  -v, --verbosity string   Log level (debug, info, warn, error, fatal, panic) (default "info")
```

