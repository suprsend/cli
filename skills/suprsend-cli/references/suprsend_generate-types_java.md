# suprsend generate-types java

Generate Java types from JSON Schema

Generate Java types from JSON Schema with specified package name

```
suprsend generate-types java [flags]
```

### Options

```
  -h, --help                 help for java
      --lombok               Generate Java Types with Lombok
      --mode string          Mode of schema to fetch (draft, live), default: live (default "live")
      --output-file string   Output file for generated Java types (default "SuprsendTypes.java")
      --package string       Package name for Java types (default "suprsend.types")
      --workspace string     Workspace to get schemas from. (default "staging")
```

### Options inherited from parent commands

```
      --config string      config file (default: $HOME/.suprsend.yaml)
  -n, --no-color           Disable color output (default: $NO_COLOR)
  -v, --verbosity string   Log level (debug, info, warn, error, fatal, panic) (default "info")
```

