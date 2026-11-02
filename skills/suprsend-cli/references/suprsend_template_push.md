# suprsend template push

Push templates and their variants from local to SuprSend workspace

Push templates and their variants from local to SuprSend workspace. Pass a slug as a positional argument or via --slug to push a single template, or omit to push all.

Local layout: each template lives in `<dir>/<slug>/` (default dir `./suprsend/templates`):

- `template.json`: `$schema`, `slug`, `name`, `description`, `tags`, `enabled_channels`, and `variant_order` (variant ids in order, keyed by `<channel>` or `<channel>/<tenant_id>` for tenant-specific ordering). Push sends `name`, `description`, `tags`, `enabled_channels` and `variant_order`; a field that is missing or `null` in the file keeps its current value (an explicit `[]` clears `enabled_channels`), and `"description": ""` clears the description. `slug` is informational: the directory name is the slug.
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

  # Commit only the variants that pass validation
  suprsend template push welcome-email --commit --skip-invalid

  # Dry run: preview what would be pushed without making changes
  suprsend template push --dry-run
```

### Tips

- Push writes to the **draft** state. Run `suprsend template commit` to promote draft → live, or pass `--commit` to push + commit in one step.
- `--dry-run` makes no API calls: it only reads the local template directories and reports the variants and channels that would be pushed. Nothing is validated server-side.
- With `--commit`, all changed variants are committed together, and one invalid variant fails the commit. `--skip-invalid` runs a pre-commit validation and commits only the variants that pass; the skipped ones are listed in the result. If every changed variant fails, nothing is committed and push exits non-zero; if nothing changed, the template is reported as `unchanged`.
- The `--commit` confirmation prompt only appears when stdin is a TTY. In non-interactive shells (CI, agents) it proceeds without asking; `-F/--force` skips it on a TTY too.
- Each template is reported on stdout as `pushed`, `committed`, `unchanged` (the draft already matched live) or `failed`, followed by a summary; with `-o json` the result is one JSON object with `items` and `counts`.

### Options

```
  -c, --commit                  Commit the pushed templates to live
      --commit-message string   Commit message describing the changes
  -d, --dir string              Input directory for templates (default: ./suprsend/templates)
  -n, --dry-run                 Report what would be pushed without making any changes
  -F, --force                   Skip the confirmation prompt for --commit
  -h, --help                    help for push
      --skip-invalid            With --commit: commit only the variants that pass pre-commit validation and list the ones skipped
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
  -w, --workspace string       Workspace slug, e.g. staging or production (default: $SUPRSEND_WORKSPACE, else the profile's workspace, else staging)
```

