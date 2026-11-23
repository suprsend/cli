# suprsend generate-types

Generate type definitions from JSON Schema

Generate typed code from trigger payload JSON schemas. Fetches schemas linked to workflows and events from a workspace and generates type definitions in the target language.

### Tips

- Run after every `suprsend schema commit` so consuming code stays in sync with the live trigger payloads.
- Pick the right subcommand for your stack: `python` (Pydantic `BaseModel`s by default, which need the `pydantic` package; `--pydantic=false` for plain dataclasses), `typescript` (optional Zod), `go`, `java`, `kotlin`, `swift`, or `dart`.
- Plain `typescript` output types `date-time`, `date` and `time` fields as `string`, which is what a JSON payload carries; `--zod` output coerces them to `Date`. A schema with no properties gives an empty type, and the command warns about it.

### Options

```
  -h, --help   help for generate-types
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
      --no-color               Disable color output (default: $NO_COLOR)
  -o, --output string          Output format: pretty, json, or yaml (default "pretty")
      --profile name           Profile to use, by name (default: $SUPRSEND_PROFILE, else the config file's active profile)
  -q, --quiet                  Suppress info and warning output (results and errors are still shown)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
```

