## suprsend sync

Sync SuprSend assets from one workspace to another

### Synopsis

Sync notification assets from one workspace to another. For each asset type, sync pulls the source workspace (`--from`, default `staging`, in `--mode`, default `live`) into a local directory (`./suprsend/<type-dir>/`, or `<dir>/<type-dir>/` with `--dir`, where `<type-dir>` is `workflows`, `schemas`, `events`, `templates`, `translations` or `preference_categories`), then pushes every asset found in that directory to the destination workspace (`--to`, default `production`) and commits it, so synced assets go live in the destination immediately. Supports syncing all asset types or a specific type (workflow, schema, event, category, translation, template). Source and destination workspaces must be different.

```
suprsend sync [flags]
```

### Examples

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

### SEE ALSO

* [suprsend](suprsend.md)	 - CLI to interact with SuprSend, a Notification Infrastructure

