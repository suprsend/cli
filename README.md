# SuprSend CLI

SuprSend CLI is a command-line interface tool for interacting with the SuprSend API, written in Go. To connect AI agents (Claude, Cursor, VS Code, Gemini, etc.) to SuprSend, use the [hosted SuprSend MCP server](https://docs.suprsend.com/reference/mcp-overview) instead of the CLI.

## What you can do with it

- **Manage workspace resources** — `template`, `workflow`, `schema`, `event`, `category`, `translation`, `workspace`. Most support `list`, `get`, `pull`, `push` and `commit` against your local working directory (`event` has no `commit`, `category` has no `get`, and `workspace` only has `list`).
- **Sync between workspaces** — `suprsend sync` pulls assets from one workspace and pushes **and commits** them in another (changes go live in the destination), using a local directory (default `./suprsend`) as intermediate storage.
- **Generate typed payload code** — `suprsend generate-types` emits typed code in Go, TypeScript, Python, Java, Kotlin, Swift, or Dart from your event / workflow JSON Schemas.
- **MCP server (removed)** — `suprsend start-mcp-server` has been removed. Use the hosted MCP server at `https://mcp.suprsend.com/mcp` (for Claude Code: `claude mcp add --transport http suprsend https://mcp.suprsend.com/mcp`), or the [suprsend/claude-plugin](https://github.com/suprsend/claude-plugin) and [suprsend/agent-plugin](https://github.com/suprsend/agent-plugin) plugins. Gemini CLI: add `"suprsend": {"httpUrl": "https://mcp.suprsend.com/mcp"}` under `mcpServers` in `~/.gemini/settings.json`, then run `/mcp auth suprsend`.
- **Multi-account / multi-environment profiles** — `suprsend profile add|modify|list|use|remove` stores per-account credentials and base URLs, so the same CLI works against public cloud, BYOC, staging, pre-prod, etc.

Full command reference: [docs.suprsend.com/reference/cli-intro](https://docs.suprsend.com/reference/cli-intro). Or run `suprsend --help` for the command tree.

## Installation

### npm / npx

Run without installing:

```bash
npx suprsend --help
```

Or install globally:

```bash
npm i -g suprsend
suprsend --help
```

Works on macOS, Linux, and Windows (x64 and arm64). Requires Node.js ≥ 18 (for `npx`).

### Homebrew

You can install SuprSend CLI using Homebrew:

```bash
brew tap suprsend/tap
brew install --cask suprsend
```

### Binary Releases

Pre-compiled binaries for various platforms are available on the [Releases page](https://github.com/suprsend/cli/releases).

### Building from Source

To build SuprSend CLI from source, follow these steps:

1. Ensure you have Go installed on your system (version 1.27.3 or later — see `go.mod`).
2. Clone the repository:
    
    ```bash
    git clone https://github.com/suprsend/cli.git
    cd cli/cmd/suprsend
    ```
    
3. Build the binary:
    
    ```bash
    go build -o suprsend
    ```
    
4. The binary will be created in the current directory. You can move it to a location in your PATH for easy access:
    
    ```bash
    sudo mv suprsend /usr/local/bin/
    ```
    
Now you can use the `suprsend` command from anywhere in your terminal.

## Usage

After installation, you can use the CLI by running the `suprsend` command. For example:

```bash
suprsend --help
```

### Using the CLI from scripts and agents

- **Target.** Commands act on the workspace from `-w`, else `SUPRSEND_WORKSPACE`, else the profile's `workspace`, else `staging`. The profile comes from `--profile`, else `SUPRSEND_PROFILE`, else the active profile (`suprsend profile use`), so each project can pin its own. The service token comes from `--service-token`, else `SUPRSEND_SERVICE_TOKEN`, else the profile. Run `suprsend whoami --check` to see the profile, host and workspace without opening the config file, and to confirm the token works and the workspace exists. Write commands also check the workspace before writing and warn when they fall back to `staging`; set `SUPRSEND_REQUIRE_WORKSPACE=1` to make that an error (and to make `sync` require `--from` and `--to`).
- **Setup without a terminal.** `suprsend profile add --name <name> --service-token <token>` (plus `--mgmnt-url`/`--base-url` for self-hosted) creates the config file if there is none and makes the first profile the active one.
- **Results.** `push`, `pull`, `commit`, `enable`, `disable`, `sync` and `generate-types` print their result on stdout even when it is piped. Push compares each item with the workspace's draft and sends only what differs. Output is one line per item (`created`, `updated`, `pushed`, `committed`, `unchanged`, `skipped`, `pulled`, `would-push`, `failed`, ...) and a summary. Add `-o json` for one JSON object with `ok`, `counts` and `items`. `-q` hides info and warning lines, not the result. `list` commands return every item unless you pass `--limit`; when that leaves items out, a note on stderr gives the total.
- **Stale copies.** `workflow` and `template` pull and push keep `.suprsend-state.json` in the asset directory; commit it with your files. Push then sends only what you changed, skips (and names) what someone changed in the workspace since your last pull, and refuses items changed on both sides. Pull again, or pass `--overwrite` to push your copy anyway. Templates are compared per part, so a dashboard edit to one variant survives a push that changed another. Files and folders whose names start with a dot are ignored.
- **Errors.** When stderr is piped or with `-o json`, an error is one JSON object on stderr (`error`, `code`, `hint`, and `details.failures` for runs over several items). Exit codes: 0 success, 1 API error (including a definition the API rejects on commit) or any failed item, 2 a local check before any API call (also when every failed item failed one), 3 auth, 4 not found, 5 rate limited, 64 invalid usage. An invalid-token error names the host the token was sent to.
- **No prompts.** Confirmation prompts only appear when stdin is a terminal, so `--commit` runs without asking in scripts. `template push|commit --skip-invalid` commits only the variants that pass validation; without it, an invalid variant fails the commit.
- **Retries.** Dropped connections and 429/502/503/504 answers are retried twice. Re-running a commit is safe: it reports `unchanged` when there is nothing left to commit.

## Documentation
Please refer to documentation [here](https://docs.suprsend.com/reference/cli-intro) OR if you want to access the cobra generated docs those are [here](docs/suprsend.md)

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

### Generating build artifacts locally

`make build` compiles the embedded type-morph Deno binary (requires [Deno](https://deno.land/)), then generates CLI documentation in `docs/` and AI skills in `skills/`.

```bash
make build
```

A CI check on PRs to `main` verifies that `docs/` and `skills/` are up to date. Run `make build` and commit the output before opening a PR. Both are generated from the command definitions, so edit the command help text rather than the generated files.

Run the unit tests (after `make build`, which creates the embedded binary the packages compile against) with:

```bash
go test ./...
```

### Removing local build artifacts
```bash
make clean
```

## License
This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
