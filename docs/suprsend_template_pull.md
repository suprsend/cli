## suprsend template pull

Pull templates and their variants from SuprSend workspace

### Synopsis

Pull templates and their variants from SuprSend workspace. Pass a slug as a positional argument or via --slug to pull a single template, or omit to pull all. Each pulled template's `<dir>/<slug>/` directory is deleted and rewritten.

Local layout: each template lives in `<dir>/<slug>/` (default dir `./suprsend/templates`):

- `template.json`: `slug`, `name`, `enabled_channels`, and `variant_order` (variant ids in order, keyed by `<channel>` or `<channel>/<tenant_id>` for tenant-specific ordering). Push sends `enabled_channels` and `variant_order`; `name` is informational and is not pushed.
- `mock_data.json`: mock data used for previews (optional).
- `<channel>/<variant_id>/variant.json`: one directory per variant.
- `<channel>/_tenants/<tenant_id>/<variant_id>/variant.json`: tenant-specific variants.
- Body files next to `variant.json`, referenced from it as `"@<file>"`: `body.raw.html`, `body.raw.txt`, `body.designer.json`, `body.designer.html`, `body.designer.txt`, `body.plain_text.txt` (email), `body.block.jsonnet` (Slack / MS Teams block kit), `body.card.jsonnet` (MS Teams jsonnet card), `body.md` (inbox). Push inlines them back into the variant.

The variant's channel, id, and tenant are read from the fields inside `variant.json`; the directory names only have to follow the shape above.

```
suprsend template pull [<slug>] [flags]
```

### Examples

```
  # Pull all templates to default directory (suprsend/templates/)
  suprsend template pull

  # Pull a single template by slug
  suprsend template pull welcome-email

  # Pull to a custom directory using the flag form
  suprsend template pull --slug welcome-email --dir ./my-templates
```

### Options

```
  -d, --dir string    Output directory for templates (default: ./suprsend/templates)
  -F, --force         Force using default directory without prompting
  -h, --help          help for pull
  -m, --mode string   Version mode: draft or live (default "live")
  -g, --slug string   Slug of a specific template to pull
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

### SEE ALSO

* [suprsend template](suprsend_template.md)	 - Manage templates

