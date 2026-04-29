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

# Clean build artifacts
make clean

# Full release build with goreleaser (generates docs, builds all platforms)
./scripts/build.sh              # snapshot build
./scripts/build.sh release      # release build
```

There are no tests in this project.

## Architecture

This is a Go CLI tool built with [cobra](https://github.com/spf13/cobra) for interacting with the SuprSend notification infrastructure API. It also serves as an MCP server for AI tool integrations.

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

### Special Features

- **`sync` command** — Pulls all assets from one workspace and pushes to another, with an optional local directory as intermediate storage
- **`generate-types` command** — Fetches JSON schemas from API and generates typed code (Python, TypeScript, Go, Java, Kotlin, Swift, Dart) using an embedded Deno binary (`type-morph/`)
- **MCP server** — `start-mcp-server` command serves tools over stdio/SSE/HTTP transports using `mcp-go`. Tools are dynamically registered for events and workflows.
  - **Default tool surface differs by entry point.** Terminal users get `--workflows=none --events=none` (minimal — only the static admin tools) so scripted invocations stay predictable. Registry-installed servers (Glama, MCP Registry) get `--workflows=all` injected via `server.json`'s `packageArguments`, so installing through a registry exposes the full trigger surface out of the box.
  - **Selector syntax**: `--workflows` accepts `all`, `none`, comma-separated slugs, or `tag:<tag>` entries (e.g. `tag:onboarding,tag:transactional`). Mixed forms work: `welcome,tag:transactional`. Tag matching is client-side — the management API does not currently support `?tags=` filtering, so all workflows are fetched and filtered in memory.
  - **Cold start**: payload-schema fetches for dynamic tools run in parallel (bounded to 10 concurrent requests). A schema fetch failure for one workflow/event no longer aborts startup — it logs and skips that single tool, leaving the rest registered.
- **Gemini CLI extension** — Built as a separate binary via goreleaser for Google Gemini CLI integration

### Release

Uses goreleaser (`.goreleaser.yaml`). Builds include macOS notarization and Homebrew cask publishing to `suprsend/homebrew-tap`. The `make build` step compiles the `type-morph` Deno binary that gets embedded into the Go binary.

After goreleaser, the release workflow also publishes to npm via `scripts/publish-npm.sh`. This ships seven packages at the same version: the unscoped root `suprsend` (a Node shim) and six platform packages `@suprsend/cli-<os>-<arch>` (darwin/linux/win32 × x64/arm64), each carrying the matching goreleaser binary. Package sources live under `npm/`. Users run `npx suprsend` or `npm i -g suprsend`.

After npm publish, the workflow publishes the MCP server metadata to the [MCP Registry](https://modelcontextprotocol.io/registry) via `mcp-publisher`. The manifest is `server.json` at the repo root and references the `suprsend` npm package; ownership is verified by the `mcpName: io.github.suprsend/cli` field in `npm/suprsend/package.json` (which must match `server.json#name`). Authentication is via GitHub OIDC — no extra secrets needed. The MCP host launches the server as `npx -y suprsend start-mcp-server` (`packageArguments` in `server.json`).

#### Pre-release tags (beta / rc / alpha / canary)

Tags with a SemVer pre-release suffix (`0.3.0-beta.1`, `0.3.0-rc.2`, `1.0.0-alpha`, etc.) flow through the same release pipeline but each downstream channel branches:

- **GitHub release** — `release.prerelease: auto` in `.goreleaser.yaml` auto-detects the suffix and marks the release as a pre-release.
- **npm** — `scripts/publish-npm.sh` derives the dist-tag from the suffix (`-beta.1` → `beta`, `-rc.2` → `rc`, `-canary.5` → `canary`) and publishes all seven packages under that dist-tag, NOT under `latest`. Stable users running `npm i suprsend` are unaffected; testers opt in via `npm i suprsend@beta`.
- **Homebrew** — `homebrew_casks[].skip_upload: auto` skips the public `suprsend/homebrew-tap` update on pre-releases. `brew upgrade suprsend` on the stable tap never bumps users to a beta build.
- **MCP Registry** — the four publish-related steps in `release.yml` are guarded by `if: ${{ !contains(github.ref_name, '-') }}`, so the registry only ever advertises stable versions.
- **Glama** — Glama releases are user-triggered through their admin UI; the maintainer chooses whether to mark a Glama release on a pre-release tag.

The detection rule across all four channels is the same: any tag with a `-` in the version string is treated as a pre-release per SemVer 2.0. If we ever need stable releases with weird suffixes, all four guards have to be revisited together.
