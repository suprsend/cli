# Template folder layout

Spec for the on-disk layout produced by `suprsend template pull` and consumed by `suprsend template push`.

Status: proposal — supersedes the layout currently implemented in `helpers.go` and `variant_files.go`. Feature not yet released; no migration concerns.

## Current layout (as of PR #50 / `template-v2`)

Implemented in:
- `internal/commands/template/helpers.go` — `WriteTemplatesToFiles` (writes `main.json`, `mock_data.json`, per-channel `variants_order.json`, and per-variant directories)
- `internal/commands/template/variant_files.go` — `writeVariantFiles` / `readAndAssembleVariant` + the `fileRefKeys` extraction table
- `internal/commands/template/template_push.go` — `readVariantOrder`, `readTemplateVariants`

Valid channels per upstream [`schema.json`](https://github.com/suprsend/schema/blob/main/template/v2/schema.json): `email`, `sms`, `whatsapp`, `inbox`, `slack`, `ms_teams`, `androidpush`, `iospush`, `webpush`.

```
suprsend/templates/
└── <template-slug>/
    ├── main.json                      # { slug, name, enabled_channels }
    ├── mock_data.json                 # optional
    └── <channel>/                     # one of the 9 channels listed above
        ├── variants_order.json        # { "ids": [variantID, ...] }
        ├── <variantID>/
        │   ├── variant.json           # variant fields; extracted fields replaced with "@<file>"
        │   ├── content.json           # always extracted (KeepRef=false: "content" key is deleted)
        │   ├── content_body_designer_design_json.json   # optional, @ref (email only)
        │   ├── content_body_designer_html.html          # optional, @ref (email only)
        │   ├── content_body_designer_text.txt           # optional, @ref (email only)
        │   ├── content_body_raw_html.html               # optional, @ref (email only)
        │   ├── content_body_raw_text.txt                # optional, @ref (email only)
        │   └── content_body_text.txt                    # optional, @ref (slack / whatsapp only)
        └── __tenant_overrides__/
            └── <tenantID>/
                ├── variants_order.json
                └── <variantID>/
                    ├── variant.json
                    └── (same extracted files as above)
```

### How the current layout works

- **`fileRefKeys` table** (`variant_files.go:25-33`) maps dot-notation JSON paths to on-disk filenames using underscore-flattened names plus an extension. Example: `content.body.designer.html` → `content_body_designer_html.html`.
- **Two extraction modes per key** (`fileRefConfig`):
  - `KeepRef=true`: the JSON key is preserved in `variant.json`, its value replaced with the string `"@<filename>"` (e.g. `"html": "@content_body_designer_html.html"`).
  - `KeepRef=false`: the JSON key is deleted from `variant.json` entirely; reassembly re-inlines the value by reading the extracted file if it exists. Currently only `content` uses this.
- **`Inline` flag** (`KeepRef=true` keys only): on push, accepts a literal value in `variant.json` if the value isn't a valid `@<file>` reference (so users can inline short strings without creating a file).
- **Tenant overrides** live in a parallel subtree rooted at `<channel>/__tenant_overrides__/<tenantID>/`, which duplicates the `variants_order.json` + `<variantID>/` structure.
- **Variant order** is stored per `(channel, tenant)` pair as `variants_order.json`.
- **Metadata** is split across `main.json` (slug/name/channels) and `mock_data.json` (mock payload).

### Pain points this spec addresses

1. Filenames like `content_body_designer_design_json.json` are hard to read and editor-unfriendly.
2. Two extraction modes (`KeepRef=true` vs `false`) and the `Inline` escape hatch make the reassembly logic non-obvious — see the three-branch decision in `readAndAssembleVariant` (`variant_files.go:196-253`).
3. `__tenant_overrides__` creates a parallel tree that duplicates the variant subtree.
4. A template with one default variant on one channel still produces 3+ JSON files and 2 directory levels before you reach the HTML/text content.
5. Template metadata is split across up to N+2 files (one `main.json`, one `mock_data.json`, one `variants_order.json` per channel per tenant).
6. The `content.body_text` extraction rule in `fileRefKeys` matches only Slack (and whatsapp) per upstream `suprsend/schema`. SMS, inbox, and push channels store their body at `content.body` as a string — their content is never extracted by the current rules, which means `content.json` (from the `content` KeepRef=false entry) ends up carrying everything. The extraction map is channel-blind where it should be channel-aware.
7. Email's `plain_text` body type (per upstream `email_body_def.plain_text`) is not extracted at all — only `raw` and `designer` have entries in `fileRefKeys`.

## Goals

- Match the JSON shape users actually edit (HTML, plain text, mock data) to file types they recognize.
- Collapse redundant nesting when only a single variant or tenant exists.
- Keep template-level metadata in one place.
- Keep the pull → edit → push round-trip predictable and idempotent.

## Proposed directory structure

```
suprsend/templates/
└── <template-slug>/
    ├── template.json                # required: template metadata + variant order
    ├── mock_data.json               # optional: mock fixture payload (any JSON shape)
    └── <channel>/                   # email | sms | whatsapp | inbox | slack | ms_teams | androidpush | iospush | webpush
        ├── <variantID>/             # one dir per variant, ALWAYS (even the single "default" case)
        │   ├── variant.json         # non-email/slack-block: content stays inline here
        │   │
        │   │   # email only (only the files matching content.body.type are expected):
        │   ├── body.raw.html
        │   ├── body.raw.txt
        │   ├── body.designer.html
        │   ├── body.designer.txt
        │   ├── body.plain_text.txt
        │   ├── design.json
        │   │
        │   │   # slack only (only when content.body_type == "block"):
        │   └── body.block.json
        └── _tenants/                # tenant overrides, same shape inside
            └── <tenantID>/
                └── <variantID>/
                    ├── variant.json
                    └── ...extracted files if any...
```

**One shape.** Every channel has variant directories — no flat / nested branching. A template with a single default variant still has `<channel>/default/`.

**Extraction is channel-specific.** Only email (HTML / design_json / plain text bodies) and Slack block bodies get extracted to sidecar files. SMS, inbox, push (android/ios/web), ms_teams, whatsapp keep their entire `content` inline in `variant.json` — their bodies are short strings that don't benefit from extraction.

## `template.json`

One file per template. Replaces `main.json` and every `variants_order.json`.

```json
{
  "$schema": "https://schema.suprsend.com/cli/template/v2/schema.json",
  "slug": "welcome",
  "name": "Welcome",
  "enabled_channels": ["email", "sms"],
  "variant_order": {
    "email":                ["default"],
    "email/tenant-xyz":     ["en", "es"],
    "sms":                  ["default"]
  }
}
```

`variant_order` lists **every** channel (and every channel + tenant pair that has overrides). Single-variant channels show `["default"]` — no absence-means-default rule. Keys for tenant overrides use the path form `<channel>/<tenantID>`, mirroring the on-disk `<channel>/_tenants/<tenantID>/` location.

## `mock_data.json` (optional)

Sits at the template root alongside `template.json`. Free-form JSON fixture used for template previews and testing. Kept separate from `template.json` because payloads can be 10–50 KB and users open it more often than other metadata.

```json
{ "user": { "name": "Alice" }, "order": { "id": "A-123", "total": "$42.00" } }
```

If absent, the template has no mock fixture — `mock_data.json` is optional on disk and optional on push.

## Variant layout rules

The rules fall into three groups: **directory & identity** (where variants live and how they're named), **content format** (what's inside `variant.json` and its sidecar files), and **push-time rules** (reconciliation and error behavior when pushing).

### Directory & identity

#### Directory shape

- Every variant lives in its own directory under `<channel>/<variantID>/`.
- Tenant overrides live under `<channel>/_tenants/<tenantID>/<variantID>/`.
- The single-default case is explicit: `<channel>/default/`.
- `_tenants/` is a reserved directory name — it cannot also be a variant ID.

#### Path is authoritative for identity fields

The variant's `id`, `channel`, and `tenant_id` are encoded both in the directory path AND as required fields in the variant payload (per upstream `variant_schema.json`). On pull, they agree. On push, **the directory path wins**:

- `channel` = immediate parent channel directory.
- `id` = variant directory name.
- `tenant_id` = `<tenantID>` if under `_tenants/`, else `null`.

The CLI overwrites the matching fields in `variant.json` from the path before sending. If they don't already match, push logs a loud warning naming both values. Renaming a directory is the supported way to rename a variant; editing the JSON field alone has no effect.

#### Why `/` in `variant_order` keys

Keys like `"email/tenant-acme"` in `template.json#variant_order` mirror the on-disk `<channel>/_tenants/<tenantID>/` path. The slash is unambiguous because channel names (fixed enum) and tenant IDs (upstream pattern `^[a-zA-Z0-9_\-]+$`) never contain `/`, so the key splits cleanly into `<channel>/<tenantID>` or a bare `<channel>`.

### Content format

#### `variant.json`

Contains the variant object with body fields extracted to sidecar files (email and Slack-block only — see extraction tables below). All other channels keep `content` intact inline.

**`variant.json` is a CLI-internal partial, not a complete API payload.** It deliberately has no `$schema` field: pointing an IDE at upstream `variant_schema.json` would light up false validation errors on email variants whose body fields have been extracted out. Validation happens inside the CLI on push, after reassembly, against the upstream schemas.

```json
{
  "id": "default",
  "channel": "email",
  "tenant_id": null,
  "locale": "en",
  "conditions": null,
  "content": {
    "subject": "Welcome!",
    "from_name": "Acme",
    "from_address": "hello@acme.com",
    "body": {
      "type": "designer"
    }
  }
}
```

Two things worth noting in this example:

- `content.body.type` remains in `variant.json` even when the body is extracted — it's the discriminator that tells push which body files to expect.
- The `content.body.designer` object is absent (upstream `email_body_def` would require `html` when `type: "designer"`). The CLI reinflates it on push by reading `body.designer.html` / `body.designer.txt` / `design.json` and populating `content.body.designer.{html,text,design_json}` in the full payload before validation.

#### Content extraction rules

Extraction applies to two specific channels; all others keep `content` inline.

##### Email (`channel: "email"`)

Driven by `content.body.type` ∈ {`raw`, `designer`, `plain_text`} (required by upstream `email_body_def`). Only files matching the active `type` are extracted; the others are ignored on push.

| Source field (JSON path)              | File                  | Active when `content.body.type` is |
|---------------------------------------|-----------------------|-------------------------------------|
| `content.body.raw.html`               | `body.raw.html`       | `raw`                               |
| `content.body.raw.text`               | `body.raw.txt`        | `raw`                               |
| `content.body.designer.html`          | `body.designer.html`  | `designer`                          |
| `content.body.designer.text`          | `body.designer.txt`   | `designer`                          |
| `content.body.designer.design_json`   | `design.json`         | `designer`                          |
| `content.body.plain_text.text`        | `body.plain_text.txt` | `plain_text`                        |

Defined in upstream [`email_schema.json`](https://github.com/suprsend/schema/blob/main/template/v2/channel/email_schema.json) (`email_body_def`, `raw_body_def`, `designer_body_def`, `plain_text_body_def`).

Fields `subject`, `preheader`, `from_name`, `from_address`, `extra_to`, `cc`, `bcc`, `reply_to`, `email_markup`, `templating_language` — all short scalars — stay inline under `content` in `variant.json`.

##### Slack (`channel: "slack"`)

| Source field (JSON path)              | File              | Active when `content.body_type` is |
|---------------------------------------|-------------------|-------------------------------------|
| `content.body_block`                  | `body.block.json` | `block`                             |

When `content.body_type == "text"`, `content.body_text` stays inline in `variant.json`.

#### Body-type discriminator is authoritative

Each channel that extracts body content uses a type field inside `variant.json` to choose which sidecar files matter. These fields differ by channel:

| Channel | Discriminator path            | Values                       |
|---------|-------------------------------|------------------------------|
| email   | `content.body.type`           | `raw` / `designer` / `plain_text` |
| slack   | `content.body_type`           | `text` / `block`             |

Rules (same for both):

- The discriminator in `variant.json` chooses which sidecar files are expected. Only matching files are read on push.
- Extra body files that don't match the active type (e.g. `body.raw.html` on disk while `type: "designer"`) are ignored on push with a warning.
- Switching types means editing the discriminator in `variant.json` and providing the new matching file(s). The CLI does not auto-delete stale body files — that's tracked as a [known limitation](#tradeoffs-and-known-limitations).

#### `body.block.json` wire format

On pull, the CLI parses the API's `body_block` string (a JSON-encoded blob) and writes it as indented JSON (2-space) in `body.block.json` for readability. On push, it re-serializes **compact (minified, no whitespace)** before injecting into `content.body_block`. This matters because `body_block` is capped at 4000 chars in upstream `slack_schema.json` — pretty-printing would eat that budget needlessly.

#### File encoding and whitespace

Body files (`body.*.html`, `body.*.txt`, `design.json`, `body.block.json`) follow a hybrid whitespace rule to keep round-trips idempotent across editors:

- **Pull** normalizes line endings to `\n` (CRLF → LF) and ensures exactly one trailing `\n`. POSIX-friendly; keeps editors with auto-newline features from producing diff churn.
- **Push** reads the file as-is, then strips at most one trailing `\n` before sending to the API.

Net effect: `pull → no edits → push` is a bit-perfect no-op on the wire even though the disk file ends with a newline the API payload doesn't.

### Push-time rules

#### `variant_order` reconciliation on push

`variant_order` in `template.json` can drift from what's on disk as users rename, add, or remove variant directories. Push reconciles with this policy:

- **New on disk, missing from `variant_order`**: auto-appended to the end of the relevant channel's (or channel+tenant's) list. Warning logged.
- **Listed in `variant_order` but missing from disk**: push errors and aborts, naming the missing `<channel>/<variantID>` path.
- **Order changes** between disk and `variant_order` are not auto-detected — `variant_order` remains the authority for ordering; rearrange it explicitly.

This keeps `mv` a first-class operation for adding a variant while protecting against silent deletion. Empty `variant_order` (`{}`) is valid — a freshly-created template with no variants shows up that way.

#### Missing body files abort push

If `variant.json` declares a body (e.g., `content.body.type: "designer"`) but the matching body file (`body.designer.html`) is absent on disk, push errors immediately and aborts the whole operation. The error message names the expected file path. Better to fail loudly than accidentally publish an empty template.

#### `enabled_channels` ↔ disk contract

Push reconciles the declared `enabled_channels` against the channel directories on disk:

- **Orphan directory** (`<channel>/` exists but `<channel>` is NOT in `enabled_channels`): push errors and aborts. You can't push to a channel the template doesn't claim.
- **Missing directory** (`<channel>` IS in `enabled_channels` but `<channel>/` doesn't exist): push logs a warning and skips that channel. Supports "template claims the channel but its variants aren't authored yet" workflows.

#### Error messages

All push-time errors are prefixed with the variant path so the offender is unambiguous:

- Missing file: `email/en: missing body.designer.html (required by content.body.type: "designer")`
- Identity mismatch: `email/en: variant.json#id is "fr" but directory is "en" — using directory value`
- Validation failure: `email/en: variant_schema violation at content.body.designer.html: <upstream message>`

Errors abort the whole push unless the rule explicitly says "warn and skip/append" (see `enabled_channels` contract and `variant_order` reconciliation).

## Schema integration

The SuprSend API payloads are already formally specified at [`suprsend/schema`](https://github.com/suprsend/schema/tree/main/template/v2). The CLI reuses those schemas directly rather than inventing parallel definitions.

### Upstream schemas (reused as-is)

| Schema                                                                                                                   | Validates                                      |
|--------------------------------------------------------------------------------------------------------------------------|------------------------------------------------|
| [`template/v2/schema.json`](https://github.com/suprsend/schema/blob/main/template/v2/schema.json)                        | Template body (`name`, `description`, `tags`, `enabled_channels`) |
| [`template/v2/variant_schema.json`](https://github.com/suprsend/schema/blob/main/template/v2/variant_schema.json)        | Variant body (`channel`, `id`, `tenant_id`, `locale`, `conditions`, `content`, …) |
| [`template/v2/channel/<channel>_schema.json`](https://github.com/suprsend/schema/tree/main/template/v2/channel)          | Per-channel `content` shape (conditionally referenced from `variant_schema.json` via `allOf` + `if/then`) |

### CLI-specific schema (`template.json` only)

`template.json` carries CLI-only fields (`slug`, `variant_order`) not present in the upstream template body. A thin wrapper composes the upstream schema and adds these:

```jsonc
// schemas/template/v2/schema.json (embedded in the CLI binary; published at
// https://schema.suprsend.com/cli/template/v2/schema.json)
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id":     "https://schema.suprsend.com/cli/template/v2/schema.json",
  "title":   "SuprSend CLI template.json",
  "allOf": [
    { "$ref": "https://schema.suprsend.com/template/v2/schema.json" }
  ],
  "type": "object",
  "properties": {
    "slug":          { "type": "string", "minLength": 1, "pattern": "^[a-zA-Z0-9_\\-.]+$" },
    "variant_order": {
      "type": "object",
      "additionalProperties": {
        "type": "array",
        "items": { "type": "string", "pattern": "^[a-zA-Z0-9_\\-.]+$" }
      }
    }
  },
  "required": ["slug", "name", "enabled_channels", "variant_order"]
}
```

`mock_data` is NOT part of `template.json` — it lives in a separate `mock_data.json` file at the template root. No CLI schema is required for it; any JSON is accepted.

### `variant.json` is deliberately unreferenced

`variant.json` does not carry a `$schema` field and is not validated by upstream `variant_schema.json` in place. The on-disk file is a **partial payload**: for email variants, body fields have been extracted to sidecar files, so `content.body.{raw,designer,plain_text}` fails the `required` constraints in the upstream channel schemas. Attaching a `$schema` reference would produce spurious red squiggles in editors and false agent signals.

Validation of the complete variant happens inside the CLI on push, against upstream `variant_schema.json`, after the CLI reassembles `content` by reading the sidecar files. Users get validation where it matters (at the API boundary) without fighting the editor during routine HTML edits.

### How the CLI uses schemas

- **Embed** both the CLI wrapper schema and a snapshot of the upstream schemas into the binary via `//go:embed` so pull/push validation works offline.
- **Publish** the CLI wrapper at `https://schema.suprsend.com/cli/template/v2/schema.json` for `$schema` editor/agent consumption. Upstream schemas already live at `schema.suprsend.com/template/v2/…` — same infra, same path convention.
- **Pull** writes `"$schema"` as the first field of each `template.json` (CLI wrapper). No `$schema` in `variant.json` — it's a partial. No `$schema` in `mock_data.json` — it's free-form.
- **Push** reassembles the full variant payload from `variant.json` + sidecar files, then validates against embedded upstream `variant_schema.json` before sending. Fast failure for bad edits.
- **Versioning.** The CLI wrapper version tracks the upstream template schema version it wraps — today that's `v2` (matching `schema.suprsend.com/template/v2/schema.json`). When upstream bumps to `v3`, the CLI ships `cli/template/v3/schema.json` at the same time. Breaking changes to the disk layout that are NOT tied to an upstream bump reuse the current version with a minor-version suffix (e.g., `v2.1`) — same path, new `$id`. The URL is the single source of truth; no parallel `layout_version` field.

### What schemas do NOT cover

Directory layout (`<channel>/<variantID>/body.designer.html`, `_tenants/`, the extraction rules in the table above) is **not** expressible in JSON Schema — schemas validate JSON documents, not filesystem trees. Layout conventions live in this document and in the filename table. Agents discover layout by `ls` + this doc; schemas validate the content of each file (post-reassembly for variants).

## Examples (current vs. proposed)

Each example shows the same template under today's layout (left) and the proposed layout (right).

### 1. Minimal email, one default variant, designer mode

<table>
<tr><th>Current</th><th>Proposed</th></tr>
<tr><td>

```
welcome/
├── main.json
└── email/
    ├── variants_order.json
    └── default/
        ├── variant.json
        ├── content.json
        └── content_body_designer_html.html
```

</td><td>

```
welcome/
├── template.json
└── email/
    └── default/
        ├── variant.json
        └── body.designer.html
```

</td></tr>
<tr><td>

`main.json`
```json
{
  "slug": "welcome",
  "name": "Welcome",
  "enabled_channels": ["email"]
}
```

`email/variants_order.json`
```json
{ "ids": ["default"] }
```

`email/default/variant.json` (content is deleted entirely; body ref lives inside content.json)
```json
{
  "id": "default",
  "channel": "email"
}
```

`email/default/content.json`
```json
{
  "subject": "Welcome!",
  "from_name": "Acme",
  "from_address": "hello@acme.com",
  "body": {
    "type": "designer",
    "designer": {
      "html": "@content_body_designer_html.html"
    }
  }
}
```

</td><td>

`template.json`
```json
{
  "$schema": "https://schema.suprsend.com/cli/template/v2/schema.json",
  "slug": "welcome",
  "name": "Welcome",
  "enabled_channels": ["email"],
  "variant_order": { "email": ["default"] }
}
```

`email/default/variant.json`
```json
{
  "id": "default",
  "channel": "email",
  "tenant_id": null,
  "locale": "en",
  "conditions": null,
  "content": {
    "subject": "Welcome!",
    "from_name": "Acme",
    "from_address": "hello@acme.com",
    "body": { "type": "designer" }
  }
}
```

</td></tr>
<tr><td>

**File count:** 5 (main + order + variant + content + body)
**Directory depth to HTML:** `welcome/email/default/` (3)
**Filenames:** underscore-flattened paths

</td><td>

**File count:** 3 (template + variant + body)
**Directory depth to HTML:** `welcome/email/default/` (3)
**Filenames:** dot-preserved, readable

</td></tr>
</table>

### 2. Email with multiple variants and a tenant override

<table>
<tr><th>Current</th><th>Proposed</th></tr>
<tr><td>

```
order-confirmed/
├── main.json
└── email/
    ├── variants_order.json
    ├── en/
    │   ├── variant.json
    │   ├── content.json
    │   └── content_body_designer_html.html
    ├── es/
    │   ├── variant.json
    │   ├── content.json
    │   └── content_body_designer_html.html
    └── __tenant_overrides__/
        └── tenant-acme/
            ├── variants_order.json
            └── en/
                ├── variant.json
                ├── content.json
                └── content_body_designer_html.html
```

</td><td>

```
order-confirmed/
├── template.json
└── email/
    ├── en/
    │   ├── variant.json
    │   └── body.designer.html
    ├── es/
    │   ├── variant.json
    │   └── body.designer.html
    └── _tenants/
        └── tenant-acme/
            └── en/
                ├── variant.json
                └── body.designer.html
```

</td></tr>
<tr><td>

`main.json`
```json
{
  "slug": "order-confirmed",
  "name": "Order confirmed",
  "enabled_channels": ["email"]
}
```

`email/variants_order.json`
```json
{ "ids": ["en", "es"] }
```

`email/__tenant_overrides__/tenant-acme/variants_order.json`
```json
{ "ids": ["en"] }
```

</td><td>

`template.json`
```json
{
  "$schema": "https://schema.suprsend.com/cli/template/v2/schema.json",
  "slug": "order-confirmed",
  "name": "Order confirmed",
  "enabled_channels": ["email"],
  "variant_order": {
    "email":             ["en", "es"],
    "email/tenant-acme": ["en"]
  }
}
```

</td></tr>
<tr><td>

**Metadata files:** 1 `main.json` + 2 `variants_order.json` = 3
**Tenant override path:** `email/__tenant_overrides__/tenant-acme/en/` (4 segments)
**Per-variant files:** 3 (variant + content + html)

</td><td>

**Metadata files:** 1 `template.json`
**Tenant override path:** `email/_tenants/tenant-acme/en/` (4 segments)
**Per-variant files:** 2 (variant + html)

</td></tr>
</table>

### 3. Multi-channel: email + SMS

<table>
<tr><th>Current</th><th>Proposed</th></tr>
<tr><td>

```
password-reset/
├── main.json
├── email/
│   ├── variants_order.json
│   └── default/
│       ├── variant.json
│       ├── content.json
│       └── content_body_designer_html.html
└── sms/
    ├── variants_order.json
    └── default/
        ├── variant.json
        └── content.json
```

</td><td>

```
password-reset/
├── template.json
├── email/
│   └── default/
│       ├── variant.json
│       └── body.designer.html
└── sms/
    └── default/
        └── variant.json
```

</td></tr>
<tr><td>

`sms/default/variant.json`
```json
{
  "id": "default",
  "channel": "sms"
}
```

`sms/default/content.json` (extracted as a whole because `content` has `KeepRef=false`)
```json
{
  "type": "basic",
  "body": "Your code is {{code}}"
}
```

</td><td>

`sms/default/variant.json` (content stays inline — SMS body is a short string)
```json
{
  "id": "default",
  "channel": "sms",
  "tenant_id": null,
  "locale": "en",
  "conditions": null,
  "needs_vendor_approval": false,
  "content": {
    "type": "basic",
    "body": "Your code is {{code}}"
  }
}
```

</td></tr>
<tr><td>

**Total files:** 8
**SMS variant files:** 2 (`variant.json` + `content.json` — split for no gain)

</td><td>

**Total files:** 4
**SMS variant files:** 1 (`variant.json` — content inline)

</td></tr>
</table>

## Benefits

- **Readable, native-editor filenames.** `body.designer.html`, `design.json`, `body.plain_text.txt` replace `content_body_designer_html.html` and friends. HTML opens in an HTML editor, JSON in a JSON editor, no `@file` indirection.
- **One shape per channel.** Every variant — including a lone default — lives under `<channel>/<variantID>/`. Pull and push walk a single rule, no branching on "is there a variants dir or not".
- **Extraction only where it helps.** Email HTML bodies and Slack Block Kit are kilobyte-scale structured payloads that benefit from their own files. SMS / inbox / push bodies are short strings that stay inline.
- **Path-authoritative identity.** Renaming a variant is `mv`. One source of truth for `id` / `channel` / `tenant_id`, so users can't desync the directory and the JSON.
- **Collapsed metadata.** One `template.json` holds slug, name, channels, and variant order — no per-channel, per-tenant `variants_order.json` files. `mock_data.json` stays separate so large fixtures don't bloat the main config.
- **Tenant overrides in a dedicated subtree.** `<channel>/_tenants/<tenantID>/<variantID>/` keeps overrides visually grouped and avoids `@` / `--` in directory names.
- **Reuses upstream schemas.** Content validation leans on the existing `suprsend/schema` definitions. The CLI only ships one new schema (`cli/template/v2/schema.json`), a thin wrapper over upstream — no parallel content schemas to maintain.

## Tradeoffs and known limitations

- **Extra directory level in the common case.** A single-default-variant template is `<channel>/default/body.designer.html` instead of `<channel>/body.designer.html`. One more `cd` for humans; imperceptible cost for tooling.
- **Less generic extraction.** Adding a new extracted field means a code change (new row in the filename table + pull/push update), not a config entry. Acceptable — the set is small, stable, and pinned to the upstream schema.
- **Identity duplication.** `id` / `channel` / `tenant_id` appear in both the path and `variant.json`. The "path wins" rule + loud warning on mismatch prevents confusion, but users need to know editing those fields in JSON is a no-op.
- **No IDE validation for `variant.json`.** Because it's a CLI-internal partial (body fields extracted to sidecars), it carries no `$schema`. Validation happens in the CLI at push time, not in the editor.
- **Stale body files on type switch.** Switching `content.body.type` from `designer` to `raw` leaves `body.designer.*` on disk. Ignored by push (warning only), but users must `rm` them manually. A `template clean` subcommand is explicitly deferred to a follow-up.
- **Order changes aren't auto-detected.** Rearranging variant directories on disk does not reorder `variant_order` in `template.json`. Additions auto-append, deletions error, but reshuffling must be done in the JSON.
- **Channel-aware pull/push logic.** Extraction rules differ between email, Slack, and everything else. Slightly more branching than a single uniform rule, but the branches are small and pinned to channel.
- **`_tenants/` is a reserved directory name.** A variant cannot be named `_tenants`. Trivial collision to document; the underscore prefix makes accidental collisions unlikely.
- **`variant_order` lists every channel.** Slightly more verbose than sparse absence-means-default, but eliminates a silent convention. Net win for discoverability.

## Required companion changes

The cross-resource spec [`../LAYOUT-artifacts.md`](../LAYOUT-artifacts.md) owns the `sync.go` changes (including adding `template` to `assetsToSync` and implementing `syncTemplates`). See its "Required companion changes" section for the full list.

## Open questions

1. **Bundled upstream copy vs. live fetch.** Embed a snapshot of upstream `suprsend/schema` at CLI build time (offline-safe, may drift) vs. lazy-fetch at validate time (always current, requires network). Lean: embed — the CLI is offline-first and upstream schemas change rarely.
