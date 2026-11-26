# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run Commands

```bash
# Build and run locally
go run ./cmd/suprsend/main.go

# Build binary
cd cmd/suprsend && go build -o suprsend

# Full build: compile type-morph binary (requires deno), generate docs and skills
make build

# Regenerate only docs/ and skills/ (what CI checks; no deno needed)
go run ./cmd/suprsend/main.go gendocs docs/
go run ./cmd/suprsend/main.go genskills skills/

# Run unit tests
go test ./...

# Clean build artifacts
make clean

# Full release build with goreleaser (generates docs, builds all platforms)
./scripts/build.sh              # snapshot build
./scripts/build.sh release      # release build
```

`go build`/`go test` need the embedded type-morph binary for your platform to exist; outside `make build`, create an empty placeholder (CI does `touch internal/utils/embedded-binaries/type-morph-linux-amd64`).

There are only a few unit tests so far (schema `--json` payload, root-type and `is_enabled` checks, `translation pull` errors, commit-result, commit status and "no changes to commit" handling, 404 messages, retries and the bulk event push reply in `mgmnt`, event push failure reporting, `--json` checks and the event pull count check, template fields and `--skip-invalid` variant selection, workspace, token and profile resolution in `config`, the `profile add|modify --workspace` flag not being resolved and being kept in the prompted `profile modify` flow, a scripted first `profile add` and `SUPRSEND_REQUIRE_WORKSPACE` in `commands`, list paging and missing-field locations in `mgmnt`, the state file and push decisions in `state`, `workflow` and `template` (against fake APIs), event push planning and the schema comparison, hidden files being skipped, `Report` output (including `-q`) in `utils`, `clierr.Failures` and `FailuresCode`, the docs/skills generators), and CI does not run them yet. The `mgmnt` tests take several seconds because they exercise the retry backoff.

### Generated docs and skills

`docs/` and `skills/suprsend-cli/` are generated from each command's `Short`/`Long`/`Example`, flag help text, and `skills:tip.*` annotations (rendered as the "Tips" list in the skill reference pages), plus the `Notes` in `internal/commands/skills.go`. Never hand-edit the generated files: change the command source and regenerate. Both generators delete their previous output first (`docs/suprsend*.md`, `skills/suprsend-cli/references/`), so pages for removed commands disappear. Hidden commands (`gendocs`, `genskills`) are excluded.

## Architecture

This is a Go CLI tool built with [cobra](https://github.com/spf13/cobra) for interacting with the SuprSend notification infrastructure API. AI tool integrations should use SuprSend's hosted MCP server (`https://mcp.suprsend.com/mcp`); the CLI's own `start-mcp-server` was removed.

### Key Layers

- **`cmd/suprsend/main.go`** — Entrypoint, calls `commands.Execute()`
- **`internal/commands/root.go`** — Root cobra command. Registers all subcommands and handles service token resolution (flag > env > config file profile), workspace resolution, and error printing
- **`internal/commands/`** — Each resource type (workflow, event, schema, category, translation, profiles) is a subpackage with its own cobra subcommands (list, pull, push, commit)
- **`mgmnt/`** — API client layer (`SS_MgmntClient`). Makes HTTP calls to SuprSend management API. Each resource type has its own file (workflow.go, event.go, schema.go, etc.)
- **`internal/config/`** — Global config singleton (`config.Cfg`) using viper. Config file is `$HOME/.suprsend.yaml`
- **`internal/utils/`** — SDK singleton (`SDKInstance`), output formatting, `Report` (results of write and pull commands), workspace existence check, embedded binaries
- **`internal/client/`** — the resty client every `mgmnt` call uses: proxy setup and retries (twice, on dropped connections and 429/502/503/504, POST and PATCH included since every management call is a read, an upsert or a commit). The workflow `trigger` path uses the suprsend-go SDK instead, so it is never retried

### Command Pattern

Each resource follows a consistent pattern:
- `helpers.go` — Shared file I/O functions (read/write JSON files)
- `<resource>.go` — Parent cobra command
- `<resource>_get.go`, `<resource>_list.go`, `<resource>_pull.go`, `<resource>_push.go`, `<resource>_commit.go` — CRUD subcommands

### Authentication

Service token resolution priority: `--service-token` flag > `SUPRSEND_SERVICE_TOKEN` env var > the profile in use. The profile in use is `--profile` > `SUPRSEND_PROFILE` > the config file's `active_profile` (`config.Cfg.Profile`, with its source; a name that isn't in the file is an error). Profiles are managed via `suprsend profile` subcommands and stored in a YAML config file (written with mode 0600). A scripted `profile add --name --service-token` creates the file when there is none and makes the first profile active. `Execute` adds the host and where it and the token came from to an invalid-token error (`explainAuth`).

### Workspace resolution

Each resource command defines its own `-w/--workspace` flag (empty default, help text `config.WorkspaceFlagHelp`). The root `PersistentPreRunE` resolves it once with `config.Cfg.ResolveWorkspace`: flag > `SUPRSEND_WORKSPACE` > the active profile's `workspace` > `staging`. It trims whitespace, rejects anything that isn't a slug, and writes the result back to the flag, so commands keep reading `cmd.Flags().GetString("workspace")`. The `profile` commands are skipped: `profile add|modify --workspace` is the default to store in the profile, not a workspace to act on. For commands named `push`, `commit`, `enable`, `disable` or `trigger` (`writeVerbs` in `root.go`) it also warns when the default was used (an error instead when `SUPRSEND_REQUIRE_WORKSPACE` is true, `config.RequireWorkspace`; `sync` then needs `--from` and `--to`) and, unless `--dry-run`, calls `utils.CheckWorkspace` so a typo fails with "unknown workspace" before anything is written. A not-found error on any other command is re-checked the same way in `Execute` (`explainNotFound`). `sync` validates `--from`/`--to` itself. `suprsend whoami` shows the resolved profile, host, workspace and token source (`--check` also verifies them against the API; when the workspace is only the built-in default it checks the token alone and says no workspace is set, and docs tell agents to pass `-w`).

### Output

Results of `push`, `pull`, `commit`, `enable`, `disable`, `sync` and `generate-types` go through `utils.Report` (`internal/utils/report.go`), never through logrus or a spinner message: logrus is silenced when stderr is piped or with `-o json`, and the spinner only draws on a terminal, so either would hide results from agents. Create one with `utils.NewReport(command, workspace, dryRun)`, call `Add(resource, status, detail)` / `Fail(resource, err)` per item (stop the spinner first), and end with `return report.Finish(err)`. Pretty mode prints a header (workspace, host, profile), one line per item and a summary to stdout whether or not it is a terminal; `-o json|yaml` prints one document from `Finish`. `utils.PushStatus` maps a `mgmnt.CommitStatus` to `pushed` / `committed` / `unchanged`. Every `push` compares each item with the workspace's draft first and sends only what differs, reporting `created` / `updated` / `unchanged` instead of `pushed`: workflows and templates through the state file (which adds `skipped`, see "State file"); schemas (`schema/push_compare.go`, the fields the API reads, a missing `is_enabled` counting as true), events (`event/push_compare.go`, by upper-cased name; only changed events go in the bulk request), categories (`category/push_compare.go`), category translations (`translation.PushLocale`) and template translations (`draftFingerprints`) by content alone. An unchanged item with `--commit` is committed with the `Finalize*` call instead. Only `sync` still reports `pushed`: it sends everything without comparing. Spinners stay for progress only: call `Stop("")`. `-q` hides the header and warnings, never the item lines or the summary. Warnings logged while logs are suppressed are collected (`config.HiddenWarnings`) and shown in the report; an explicit `-v` (or `DEBUG`) still logs to stderr when it is piped.

`list` commands return every item by default (`--limit 0`): the `mgmnt` list functions page through with `listPages` (`mgmnt/pages.go`), which stops at the API's `meta.count` rather than at a short page, and return that total in `Meta.Count`. When `--limit` leaves items out, `utils.NoteMoreResults` says so on stderr, so `-o json` stays a plain array.

The `mgmnt` functions that can commit (`PushWorkflow`, `FinalizeWorkflow`, `PushSchema`, `FinalizeSchema`, `PushCategories`, `FinalizeCategories`, `FinalizeTranslation`, `CommitTemplate`) return `(CommitStatus, error)`: `Committed`, `NoChanges` (draft already matched live, not an error) or `NotCommitted`. A commit request that gets no response after retries returns a "commit status unknown" error (`requestError`).

### State file

`internal/state` keeps `.suprsend-state.json` in the workflows and templates directories: per workspace and item, a fingerprint of the workspace's version and of the local copy at the last pull or push (`state.Fingerprint` leaves out server-managed fields: `workflowIgnore`, `variantIgnore`). Pull records what it wrote (`workflow.RecordPulled`, `template.RecordPulled`, which reads the files back the way push does). Push fetches the draft, and `state.Decide` picks create / update / unchanged / skip (changed only in the workspace) / conflict (changed on both sides; `--overwrite` pushes anyway). Skipped and conflicting items keep their old entry so they stay detected until the next pull. The recorded workspace fingerprint is always the **draft's**, since that is what push compares with: a pull in live mode fetches the draft too (`draftsBySlug`, `FetchOptions.WithDraft`) and sets `Entry.Pending` when it differs from what was pulled (or the part is missing from the draft). `Decide` returns `Pending` for such an item while neither side has changed (reported `unchanged` with a note; never sent, never committed) and `Conflict` once the local copy changes. Workflow fingerprints sort the `channels` of `send_multi_channel` nodes (`sortChannels`): the API keeps them as a set and returns draft and live in different orders. Schemas have no state file; `pushSchema` instead leaves the draft alone when the file matches live and the draft differs (`matchesLive`). Categories do the same against the live tree (a category push replaces the whole draft tree). Because of that replace, `category push` and `sync` name the categories they remove (`category.RemovedCategories`). Categories are served by `preferences-service`, not the main API: it keeps a category's `category` slug and a section's `slug` when given and derives them from the name otherwise, so `mgmnt.Section.Slug` must round-trip through `categories.json` (section translations are keyed by it).

Comparisons follow what the API stores, checked against the API source: the workflow fingerprint sorts every node property the API keeps as a set (`setFields`) and `trigger_events`; `variantIgnore` includes `seq_no` (renumbered when a sibling variant changes); mock data is per version, so `GetTemplateMockData` takes a mode; the schema comparison (`sameAs`) treats a missing `additionalProperties` as `false` when `properties` is set and takes a `name`/`description` the payload leaves out from the workspace; an event's `payload_schema` is normalised (`schemaLink`: null when there is no slug, `version_no` null when unpinned); draft translation files marked `action: deleted` count as missing. When an event's schema is removed, `event push` calls `DelinkEventSchema` first, because the bulk endpoint leaves the link record behind when no event in the request has a schema. A validation error's `message` can be an object or a list; `extractMessage` flattens it. Templates are decided per part (`fields`, `variants/<channel>/<id>`, `mock_data`, `variant_order`; `template/push_state.go`), and the push-side fetch (`fetchDraft`) fails on any request error rather than treating a part as missing. The `fields` part is always fingerprinted over all of `mgmnt.TemplateFields` (a field the file leaves out counts as null), so adding or removing a key in `template.json` reads as a local change, not a conflict. A commit publishes the whole draft, so `--commit` fails when the draft has a variant that isn't in the local files and is new or differs from live (`unseenDraftVariants`); `--overwrite` commits it and names it in the result. `sync` passes `PushAll` and keeps no state. The file is meant to be committed; a missing file just means no record. Every reader of asset directories uses `utils.ReadDir` / `utils.IsHidden`, which skip dot files, so the state file (and `.DS_Store`, `.git`) is never read as an asset.

### Error Handling

Use `clierr` (`internal/clierr/clierr.go`) for all user-facing errors — never `fmt.Errorf` or `errors.New` at command boundaries.

- `clierr.New(message, code)` — create a new structured error
- `clierr.Wrap(err, code, hint)` — promote a plain error with a code and hint
- `clierr.Failures(summary, failures, code)` — the error for a run over several items where some failed; lists each failure in the message and in `details.failures`, since per-item log lines are hidden when stderr is piped. Build each entry with `clierr.ItemFailure(resource, action, err)`, which avoids repeating a resource prefix the error already has
- Codes are constants in `clierr` (e.g. `CodeConfigInvalid`, `CodeAuthMissingToken`, `CodeFileParseFailed`). Pick the closest match; use `CodeUnknown` as a last resort. Exit code 2 is only for checks the CLI runs before calling the API: a definition the API rejects is `CodeAPIValidation` (exit 1), not `CodeSchemaValidation`.
- A run over several items picks its code with `clierr.FailuresCode(errs)`: exit 2 when every failure was a local check, else 1. Wrap local read/parse errors with a local code (`CodeFileParseFailed`, `CodeFileNotFound`) before recording them so this works.
- `clierr` errors carry a numeric exit code (auth=3, validation=2, not-found=4, etc.) and render correctly in both plain-text and `--output json` modes. `Execute` prints every error itself (`utils.WriteError`): "Error: <message>" plus "Hint: <hint>" in plain text, one JSON object on stderr when stderr is piped or with `-o json`. Cobra's own error printing is always silenced.
- Don't also `log.Error` an error you return, and don't list failed items in log lines: the report marks each failed item and the returned error carries the reasons, so each failure is printed once.

### Special Features

- **`sync` command** — Pulls all assets from one workspace into a local directory (default `./suprsend`, override with `--dir`) and pushes everything in that directory to another workspace. Commit is hardcoded on (`commit := true`), so every synced asset goes live in the destination; `--dry-run` still pulls and overwrites the local files. Templates are synced with `--skip-invalid` semantics (only variants that pass pre-commit validation are committed)
- **`template push|commit --skip-invalid`** — commits only the changed variants that pass pre-commit validation and names the skipped ones. Without it every variant is committed and an invalid one fails the commit. `-F/--force` only skips the confirmation prompt (it used to imply the partial commit too). `template push` sends `name`, `description`, `tags` and `enabled_channels` from `template.json` (a missing or `null` key keeps the current value; before, a missing `enabled_channels` was sent as `null` and cleared the draft's channels), and `template pull` writes them with `$schema: https://schema.suprsend.com/template/v2/schema.json`. After pushing `mock_data.json` it warns if the API stored `recipient_distinct_id`, `actor_distinct_id` or `tenant_id` differently from the file (the API drops some silently)
- **`event pull`** — lists every event; it no longer passes `has_linked_schema=true`, which filtered out events without a linked payload schema. It fails if it writes fewer events than the API's `meta.count`.
- **Interactive prompts** — `profile add` / `profile modify` use a plain stdin prompt helper (`ask` in `internal/commands/profiles/helpers.go`). Don't bring back a bubbletea-based UI library: bubbletea's `init()` queries the terminal (OSC 11 and cursor-position escape sequences) on every run of the binary, which corrupted agents' captured output
- **`generate-types` command** — Fetches JSON schemas from API and generates typed code (Python, TypeScript, Go, Java, Kotlin, Swift, Dart) using an embedded Deno binary (`type-morph/`). The generator's stdout goes to the debug log and its stderr into the returned error, so only the report reaches stdout. `type-morph/main.ts` strips date formats for plain `typescript` (dates are typed `string`; `--zod` keeps them). A change to `type-morph/` only takes effect once the embedded binaries are rebuilt (`make build`, needs deno)
- **MCP server (removed)** — `start-mcp-server` (stdio/SSE/HTTP via `mcp-go`, with `internal/tools/`) was deprecated in favour of the hosted MCP server (`https://mcp.suprsend.com/mcp`, OAuth sign-in) and the `suprsend/claude-plugin` / `suprsend/agent-plugin` plugins, then removed. Don't bring it back; point AI tool integrations at the hosted server.
- **Retired distribution channels** — the MCP Registry entry `io.github.suprsend/cli` (now deprecated; the hosted server is listed as `com.suprsend/mcp`), the Gemini CLI extension (`gemini-extension.json` and its goreleaser build/archive) and Glama (`glama.json` and the README badge; the listing on glama.ai itself is unlisted by hand in Glama's admin UI) were retired along with `start-mcp-server`. Gemini CLI users connect to the hosted server via `httpUrl` in `~/.gemini/settings.json`.

### Release

Uses goreleaser (`.goreleaser.yaml`). Builds include macOS notarization and Homebrew cask publishing to `suprsend/homebrew-tap`. The `make build` step compiles the `type-morph` Deno binary that gets embedded into the Go binary.

After goreleaser, the release workflow also publishes to npm via `scripts/publish-npm.sh`. This ships seven packages at the same version: the unscoped root `suprsend` (a Node shim) and six platform packages `@suprsend/cli-<os>-<arch>` (darwin/linux/win32 × x64/arm64), each carrying the matching goreleaser binary. Package sources live under `npm/`. Users run `npx suprsend` or `npm i -g suprsend`.

#### Pre-release tags (beta / rc / alpha / canary)

Tags with a SemVer pre-release suffix (`0.3.0-beta.1`, `0.3.0-rc.2`, `1.0.0-alpha`, etc.) flow through the same release pipeline but each downstream channel branches:

- **GitHub release** — `release.prerelease: auto` in `.goreleaser.yaml` auto-detects the suffix and marks the release as a pre-release.
- **npm** — `scripts/publish-npm.sh` derives the dist-tag from the suffix (`-beta.1` → `beta`, `-rc.2` → `rc`, `-canary.5` → `canary`) and publishes all seven packages under that dist-tag, NOT under `latest`. Stable users running `npm i suprsend` are unaffected; testers opt in via `npm i suprsend@beta`.
- **Homebrew** — `homebrew_casks[].skip_upload: auto` skips the public `suprsend/homebrew-tap` update on pre-releases. `brew upgrade suprsend` on the stable tap never bumps users to a beta build.

The detection rule across all of these is the same: any tag with a `-` in the version string is treated as a pre-release per SemVer 2.0. If we ever need stable releases with weird suffixes, all of these guards have to be revisited together.
