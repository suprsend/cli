# suprsend schema push

Push schemas

Upload local schemas to a workspace. For every subdirectory `<dir>/<slug>/` of the input directory (default `./suprsend/schemas`), reads `schema.json` (name, description, and `is_enabled` if present) and `payload_schema.json` (the JSON Schema), validates the payload schema locally, and pushes both; the directory name is used as the slug. By default, changes are staged as drafts. Use --commit to also promote to live. Pass a slug as a positional argument or via --slug to push a single schema, or omit to push all. With a slug, --json takes the JSON Schema inline instead of reading the slug's files.

```
suprsend schema push [<slug>] [flags]
```

## Examples

```
  # Push all schemas from default directory
  suprsend schema push

  # Push a single schema and commit to live immediately
  suprsend schema push order-placed --commit

  # Push an inline JSON Schema instead of reading payload_schema.json
  suprsend schema push order-placed --json '{"type":"object","properties":{"order_id":{"type":"string"}},"required":["order_id"]}'

  # Dry run: preview what would be pushed without making changes
  suprsend schema push --dry-run
```

### Tips

- Push writes to the **draft** state. Run `suprsend schema commit` to promote draft → live, or pass `--commit` to push + commit in one step.
- `--dry-run` makes no API calls. It reads the local files, compiles each payload schema (`payload_schema.json`, or the `--json` value) locally as JSON Schema (draft 2020-12; root `type` must be `object`), and reports what would be pushed. Nothing is validated server-side.
- After committing a schema change, regenerate types with `suprsend generate-types <language>` so consuming code stays in sync.
- With `--commit`, a schema that is saved as a draft but not committed (for example an empty JSON Schema) is an error: the command exits non-zero with the API's reason. A schema whose draft already matches live counts as success (nothing to commit).
- `-F/--force` only skips the confirmation prompt for `--commit`; it does not change what is committed. The prompt only appears when stdin is a TTY, so in non-interactive shells (CI, agents) `--commit` proceeds without asking even without `-F`.
- Enabled/disabled is a property of the schema, not a version: it applies on push, without `--commit`. If `schema.json` has no `is_enabled` key, the API currently **enables** the schema on every push, re-enabling a disabled one (this includes `sync`). To keep a schema disabled, set `"is_enabled": false` in its `schema.json`. `--json` pushes never send `is_enabled`.
- Each schema is compared with the workspace's draft (name, description, JSON Schema and `is_enabled`) and reported on stdout as `created` or `updated` (saved as draft), `unchanged` (the draft already matches, so nothing was sent; with `--commit`, the draft already matched live), `committed` or `failed`, followed by a summary; with `-o json` the result is one JSON object with `items` and `counts`. A file that matches the live version while the draft differs is reported `unchanged` with a note, and the draft is left alone: it is pending work that a pull without `--mode draft` didn't bring, and pushing would replace it with what is already live.

### Options

```
  -c, --commit                  Promote changes from draft to live after pushing
      --commit-message string   Message describing the changes being committed
  -d, --dir string              Directory containing schema files (default: ./suprsend/schemas)
  -n, --dry-run                 Report what would be pushed without making any changes
  -F, --force                   Skip confirmation prompt when --commit is set
  -h, --help                    help for push
  -j, --json string             Payload JSON Schema as a JSON object, used instead of payload_schema.json (requires --slug). Must be a valid JSON Schema with root type "object", e.g. '{"type":"object","properties":{"key":{"type":"string"}}}'
  -g, --slug string             Schema slug to push (omit to push all)
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
  -w, --workspace string       Workspace slug, e.g. staging or production (default: $SUPRSEND_WORKSPACE, else the profile's workspace, else staging)
```

