# suprsend sync

Sync SuprSend assets from one workspace to another

Sync notification assets from one workspace to another. Pulls assets from the source workspace and pushes them to the destination. Supports syncing all asset types or a specific type (workflow, schema, event, category, translation, template). Source and destination workspaces must be different.

```
suprsend sync [flags]
```

## Examples

```
  # Sync all assets from staging to production
  suprsend sync --from staging --to production

  # Sync only workflows
  suprsend sync --from staging --to production --assets workflow

  # Sync and commit immediately (prompts for confirmation)
  suprsend sync --from staging --to production --commit

  # Dry run: preview what would be synced without making changes
  suprsend sync --from staging --to production --dry-run
```

### Tips

- `--from` is the source, `--to` is the destination. They must be different workspaces; sync **overwrites** drafts in the destination.
- Pair with `--dry-run` to validate every asset server-side without writing to the destination. Add `--assets <type>` to scope to one resource type (workflow / schema / event / category / translation / template).

### Options

```
  -a, --assets string           Asset types to sync: all, workflow, schema, event, category, translation, or template (default "all")
  -c, --commit                  Promote changes from draft to live after syncing
      --commit-message string   Commit message applied to every committed resource in this sync run (required when --commit is set)
  -d, --dir string              Local directory for intermediate file storage during sync
  -n, --dry-run                 Print what would be synced without making any changes
  -F, --force                   Skip confirmation prompt
  -S, --from string             Source workspace to pull assets from (default "staging")
  -h, --help                    help for sync
  -m, --mode string             Version mode: draft or live (default "live")
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

