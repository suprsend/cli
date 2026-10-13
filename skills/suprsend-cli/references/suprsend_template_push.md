# suprsend template push

Push templates and their variants from local to SuprSend workspace

Push templates and their variants from local to SuprSend workspace. Pass a slug as a positional argument or via --slug to push a single template, or omit to push all.

Local layout: each template lives in `<dir>/<slug>/` (default dir `./suprsend/templates`):

- `template.json`: `slug`, `name`, `enabled_channels`, and `variant_order` (variant ids in order, keyed by `<channel>` or `<channel>/<tenant_id>` for tenant-specific ordering). Push sends `enabled_channels` and `variant_order`; `name` is informational and is not pushed.
- `mock_data.json`: mock data used for previews (optional).
- `<channel>/<variant_id>/variant.json`: one directory per variant.
- `<channel>/_tenants/<tenant_id>/<variant_id>/variant.json`: tenant-specific variants.
- Body files next to `variant.json`, referenced from it as `"@<file>"`: `body.raw.html`, `body.raw.txt`, `body.designer.json`, `body.designer.html`, `body.designer.txt`, `body.plain_text.txt` (email), `body.block.jsonnet` (Slack / MS Teams block kit), `body.card.jsonnet` (MS Teams jsonnet card), `body.md` (inbox). Push inlines them back into the variant.

The variant's channel, id, and tenant are read from the fields inside `variant.json`; the directory names only have to follow the shape above.

```
suprsend template push [<slug>] [flags]
```

## Examples

```
  # Push all templates from default directory
  suprsend template push

  # Push a single template and commit to live immediately
  suprsend template push welcome-email --commit

  # Dry run: preview what would be pushed without making changes
  suprsend template push --dry-run
```

### Tips

- Push writes to the **draft** state. Run `suprsend template commit` to promote draft → live, or pass `--commit` to push + commit in one step.
- `--dry-run` makes no API calls: it only reads the local template directories and logs the variants and channels that would be pushed. Nothing is validated server-side.
- With `--commit`, `-F/--force` does two things: it skips the confirmation prompt **and** switches to a partial commit. The CLI runs a pre-commit validation and commits only the variants that pass; invalid variants are skipped with a warning. If every changed variant fails, nothing is committed and push exits non-zero; if nothing changed, push succeeds without committing. Without `-F`, all variants are committed together.
- The `--commit` confirmation prompt only appears when stdin is a TTY. In non-interactive shells (CI, agents) it proceeds without asking, even without `-F`.

### Options

```
  -c, --commit                  Commit the pushed templates to live
      --commit-message string   Commit message describing the changes
  -d, --dir string              Input directory for templates (default: ./suprsend/templates)
  -n, --dry-run                 Print what would be pushed without making any changes
  -F, --force                   With --commit: skip the confirmation prompt and commit only the variants that pass pre-commit validation (invalid variants are skipped)
  -h, --help                    help for push
  -g, --slug string             Slug of a specific template to push
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
      --no-color               Disable color output (default: $NO_COLOR)
  -o, --output string          Output format: pretty, json, or yaml (default "pretty")
  -q, --quiet                  Suppress info/warn output (errors are still shown)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -w, --workspace string       Workspace to list templates from (default "staging")
```

