# suprsend generate-types java

Generate Java types from JSON Schema

Generate Java type definitions from trigger payload schemas. Creates one Java file per linked schema in a package directory structure. Supports Lombok annotations with --lombok.

```
suprsend generate-types java [flags]
```

### Options

```
  -h, --help                 help for java
      --lombok               Add Lombok annotations to generated classes
      --mode string          Version mode: draft or live (default "live")
      --output-file string   Output file path for generated types (default "SuprsendTypes.java")
      --package string       Java package name for generated classes (default "suprsend.types")
      --workspace string     Workspace name (e.g., staging, production) (default "staging")
```

### Options inherited from parent commands

```
      --config string      config file (default: $HOME/.suprsend.yaml)
  -n, --no-color           Disable color output (default: $NO_COLOR)
  -v, --verbosity string   Log level (debug, info, warn, error, fatal, panic) (default "info")
```

