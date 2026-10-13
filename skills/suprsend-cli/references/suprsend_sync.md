# suprsend sync

Sync SuprSend assets from one workspace to another

Sync notification assets from one workspace to another. For each asset type, sync pulls the source workspace (`--from`, default `staging`, in `--mode`, default `live`) into a local directory (`./suprsend/<type-dir>/`, or `<dir>/<type-dir>/` with `--dir`, where `<type-dir>` is `workflows`, `schemas`, `events`, `templates`, `translations` or `preference_categories`), then pushes every asset found in that directory to the destination workspace (`--to`, default `production`) and commits it, so synced assets go live in the destination immediately. Supports syncing all asset types or a specific type (workflow, schema, event, category, translation, template). Source and destination workspaces must be different.

```
suprsend sync [flags]
```

## Examples

```
  # Sync all assets from staging to production (pushes and commits to live)
  suprsend sync --from staging --to production

  # Sync only workflows
  suprsend sync --from staging --to production --assets workflow

  # Dry run: pull staging into local files and list what would be pushed (destination untouched)
  suprsend sync --from staging --to production --dry-run

  # Sync through a fresh directory so only assets that exist in staging are pushed
  suprsend sync --from staging --to production --dir ./sync-tmp
```

### Tips

- `--from` is the source (default `staging`), `--to` is the destination (default `production`); they must be different workspaces. Sync pushes **and commits** every asset type, so changes go **live** in the destination immediately. There is no draft-only sync.
- Committing is per asset: a workflow or schema that is saved as a draft but not committed (for example a workflow that fails validation) counts as a failure, and sync exits non-zero after trying the rest. Template variants that fail pre-commit validation are skipped while the valid ones are committed; the skips are only logged as warnings, and a template with no valid changed variant counts as a failure. Events the destination rejects (typically a `payload_schema` pinned to a schema `version_no` that does not exist there, since each workspace numbers schema versions separately) are listed and count as a failure; the other events are still saved. Assets with nothing to commit (the destination already matches) count as success, so re-running sync is safe.
- Sync pushes **every** asset in the local sync directory, not just what it pulled: local-only workflows, schemas, events, templates or translation files under `./suprsend/{workflows,schemas,events,templates,translations}/` are pushed and committed too. Categories are the exception: `categories.json` is overwritten from the source. Pulled assets overwrite local files with the same slug. Use a fresh `--dir` to sync exactly what exists in the source.
- `--dry-run` makes no calls to the destination, but it still pulls the source and **overwrites local files** in the sync directory (template slug directories are deleted and rewritten). Only local checks run (JSON parsing, plus a JSON Schema compile for schemas); nothing is validated server-side. Add `--assets <type>` to scope to one resource type (workflow / schema / event / category / translation / template).
- The confirmation prompt only appears when stdin is a TTY. In non-interactive shells (CI, agents) sync starts immediately, even without `--force`.

### Options

```
  -a, --assets string           Asset types to sync: all, workflow, schema, event, category, translation, or template (default "all")
      --commit-message string   Commit message applied to every committed resource in this sync run
  -d, --dir string              Local directory for intermediate file storage; each asset type uses a subdirectory (default: ./suprsend)
  -n, --dry-run                 Pull the source into local files, then only print what would be pushed (no changes to the destination)
  -F, --force                   Skip confirmation prompt
  -S, --from string             Source workspace to pull assets from (default "staging")
  -h, --help                    help for sync
  -m, --mode string             Version of the source assets to pull: draft or live (default "live")
  -t, --to string               Destination workspace to push assets to (default "production")
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
      --no-color               Disable color output (default: $NO_COLOR)
  -o, --output string          Output format: pretty, json, or yaml (default "pretty")
  -q, --quiet                  Suppress info/warn output (errors are still shown)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
```

