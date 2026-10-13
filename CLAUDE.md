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

There are only a few unit tests so far (schema `--json` payload, root-type and `is_enabled` checks, `translation pull` errors, commit-result and "no changes to commit" handling plus the bulk event push reply in `mgmnt`, event push failure reporting and `--json` checks, `clierr.Failures`, the docs/skills generators), and CI does not run them yet.

### Generated docs and skills

`docs/` and `skills/suprsend-cli/` are generated from each command's `Short`/`Long`/`Example`, flag help text, and `skills:tip.*` annotations (rendered as the "Tips" list in the skill reference pages), plus the `Notes` in `internal/commands/skills.go`. Never hand-edit the generated files: change the command source and regenerate. Both generators delete their previous output first (`docs/suprsend*.md`, `skills/suprsend-cli/references/`), so pages for removed commands disappear. Hidden commands (`gendocs`, `genskills`) are excluded.

## Architecture

This is a Go CLI tool built with [cobra](https://github.com/spf13/cobra) for interacting with the SuprSend notification infrastructure API. It also contains a deprecated MCP server (`start-mcp-server`); AI tool integrations should use SuprSend's hosted MCP server instead.

### Key Layers

- **`cmd/suprsend/main.go`** — Entrypoint, calls `commands.Execute()`
- **`internal/commands/root.go`** — Root cobra command. Registers all subcommands and handles service token resolution (env > flag > config file profile)
- **`internal/commands/`** — Each resource type (workflow, event, schema, category, translation, profiles) is a subpackage with its own cobra subcommands (list, pull, push, commit)
- **`mgmnt/`** — API client layer (`SS_MgmntClient`). Makes HTTP calls to SuprSend management API. Each resource type has its own file (workflow.go, event.go, schema.go, etc.)
- **`internal/tools/`** — MCP tool definitions. Each file registers tools via `RegisterTool()` into a global registry. The MCP server (`startMcpServer.go`) selects and serves these tools
- **`internal/config/`** — Global config singleton (`config.Cfg`) using viper. Config file is `$HOME/.suprsend.yaml`
- **`internal/utils/`** — SDK singleton (`SDKInstance`), output formatting, embedded binaries

### Command Pattern

Each resource follows a consistent pattern:
- `helpers.go` — Shared file I/O functions (read/write JSON files)
- `<resource>.go` — Parent cobra command
- `<resource>_get.go`, `<resource>_list.go`, `<resource>_pull.go`, `<resource>_push.go`, `<resource>_commit.go` — CRUD subcommands

### Authentication

Service token resolution priority: `SUPRSEND_SERVICE_TOKEN` env var > `--service-token` flag > active profile in config file. Profiles are managed via `suprsend profile` subcommands and stored in a YAML config file.

### Error Handling

Use `clierr` (`internal/clierr/clierr.go`) for all user-facing errors — never `fmt.Errorf` or `errors.New` at command boundaries.

- `clierr.New(message, code)` — create a new structured error
- `clierr.Wrap(err, code, hint)` — promote a plain error with a code and hint
- `clierr.Failures(summary, failures, code)` — the error for a run over several items where some failed; lists each failure in the message and in `details.failures`, since per-item log lines are hidden when stderr is piped. Build each entry with `clierr.ItemFailure(resource, action, err)`, which avoids repeating a resource prefix the error already has
- Codes are constants in `clierr` (e.g. `CodeConfigInvalid`, `CodeAuthMissingToken`, `CodeFileParseFailed`). Pick the closest match; use `CodeUnknown` as a last resort.
- `clierr` errors carry a numeric exit code (auth=3, validation=2, not-found=4, etc.) and render correctly in both plain-text and `--output json` modes.

### Special Features

- **`sync` command** — Pulls all assets from one workspace into a local directory (default `./suprsend`, override with `--dir`) and pushes everything in that directory to another workspace. Commit is hardcoded on (`commit := true`), so every synced asset goes live in the destination; `--dry-run` still pulls and overwrites the local files
- **`generate-types` command** — Fetches JSON schemas from API and generates typed code (Python, TypeScript, Go, Java, Kotlin, Swift, Dart) using an embedded Deno binary (`type-morph/`)
- **MCP server (deprecated)** — `start-mcp-server` command serves tools over stdio/SSE/HTTP transports using `mcp-go`. Tools are dynamically registered for events and workflows.
  - **Deprecated in favour of the hosted MCP server** (`https://mcp.suprsend.com/mcp`, OAuth sign-in) and the `suprsend/claude-plugin` / `suprsend/agent-plugin` plugins. Cobra's `Deprecated` field prints the notice on stderr (stdout stays clean JSON-RPC for stdio clients) and hides the command from help, completion and the generated docs/skills; the server also sends MCP `instructions` pointing assistants to the replacement. It stays only for service-token auth and BYOC instances, which the hosted server doesn't cover yet, so don't add features to it.
  - **Default tool surface differs by entry point.** Terminal users get `--workflows=none --events=none` (minimal — only the static admin tools) so scripted invocations stay predictable. Registry-installed servers (Glama, MCP Registry) get `--workflows=all` injected via `server.json`'s `packageArguments`, so installing through a registry exposes the full trigger surface out of the box.
  - **Selector syntax**: `--workflows` accepts `all`, `none`, comma-separated slugs, or `tag:<tag>` entries (e.g. `tag:onboarding,tag:transactional`). Mixed forms work: `welcome,tag:transactional`. Tag matching is client-side — the management API does not currently support `?tags=` filtering, so all workflows are fetched and filtered in memory.
  - **Cold start**: payload-schema fetches for dynamic tools run in parallel (bounded to 10 concurrent requests). A schema fetch failure for one workflow/event no longer aborts startup — it logs and skips that single tool, leaving the rest registered.
- **Gemini CLI extension** — Built as a separate binary via goreleaser for Google Gemini CLI integration

### Release

Uses goreleaser (`.goreleaser.yaml`). Builds include macOS notarization and Homebrew cask publishing to `suprsend/homebrew-tap`. The `make build` step compiles the `type-morph` Deno binary that gets embedded into the Go binary.

After goreleaser, the release workflow also publishes to npm via `scripts/publish-npm.sh`. This ships seven packages at the same version: the unscoped root `suprsend` (a Node shim) and six platform packages `@suprsend/cli-<os>-<arch>` (darwin/linux/win32 × x64/arm64), each carrying the matching goreleaser binary. Package sources live under `npm/`. Users run `npx suprsend` or `npm i -g suprsend`.

After npm publish, the workflow publishes the (deprecated) CLI MCP server metadata to the [MCP Registry](https://modelcontextprotocol.io/registry) via `mcp-publisher`. The manifest is `server.json` at the repo root and references the `suprsend` npm package; ownership is verified by the `mcpName: io.github.suprsend/cli` field in `npm/suprsend/package.json` (which must match `server.json#name`). Authentication is via GitHub OIDC — no extra secrets needed. The MCP host launches the server as `npx -y suprsend start-mcp-server` (`packageArguments` in `server.json`).

#### Pre-release tags (beta / rc / alpha / canary)

Tags with a SemVer pre-release suffix (`0.3.0-beta.1`, `0.3.0-rc.2`, `1.0.0-alpha`, etc.) flow through the same release pipeline but each downstream channel branches:

- **GitHub release** — `release.prerelease: auto` in `.goreleaser.yaml` auto-detects the suffix and marks the release as a pre-release.
- **npm** — `scripts/publish-npm.sh` derives the dist-tag from the suffix (`-beta.1` → `beta`, `-rc.2` → `rc`, `-canary.5` → `canary`) and publishes all seven packages under that dist-tag, NOT under `latest`. Stable users running `npm i suprsend` are unaffected; testers opt in via `npm i suprsend@beta`.
- **Homebrew** — `homebrew_casks[].skip_upload: auto` skips the public `suprsend/homebrew-tap` update on pre-releases. `brew upgrade suprsend` on the stable tap never bumps users to a beta build.
- **MCP Registry** — the four publish-related steps in `release.yml` are guarded by `if: ${{ !contains(github.ref_name, '-') }}`, so the registry only ever advertises stable versions.
- **Glama** — Glama releases are user-triggered through their admin UI; the maintainer chooses whether to mark a Glama release on a pre-release tag.

The detection rule across all four channels is the same: any tag with a `-` in the version string is treated as a pre-release per SemVer 2.0. If we ever need stable releases with weird suffixes, all four guards have to be revisited together.
