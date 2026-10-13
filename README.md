# SuprSend CLI

SuprSend CLI is a command-line interface tool for interacting with the SuprSend API, written in Go. To connect AI agents (Claude, Cursor, VS Code, Gemini, etc.) to SuprSend, use the [hosted SuprSend MCP server](https://docs.suprsend.com/reference/mcp-overview) instead of the CLI.

## What you can do with it

- **Manage workspace resources** — `template`, `workflow`, `schema`, `event`, `category`, `translation`, `workspace`. Most support `list`, `get`, `pull`, `push` and `commit` against your local working directory (`event` has no `commit`, `category` has no `get`, and `workspace` only has `list`).
- **Sync between workspaces** — `suprsend sync` pulls assets from one workspace and pushes **and commits** them in another (changes go live in the destination), using a local directory (default `./suprsend`) as intermediate storage.
- **Generate typed payload code** — `suprsend generate-types` emits typed code in Go, TypeScript, Python, Java, Kotlin, Swift, or Dart from your event / workflow JSON Schemas.
- **MCP server (deprecated)** — `suprsend start-mcp-server` still works but is deprecated. Use the hosted MCP server at `https://mcp.suprsend.com/mcp` (for Claude Code: `claude mcp add --transport http suprsend https://mcp.suprsend.com/mcp`), or the [suprsend/claude-plugin](https://github.com/suprsend/claude-plugin) and [suprsend/agent-plugin](https://github.com/suprsend/agent-plugin) plugins. Gemini CLI: add `"suprsend": {"httpUrl": "https://mcp.suprsend.com/mcp"}` under `mcpServers` in `~/.gemini/settings.json`, then run `/mcp auth suprsend` (the Gemini CLI extension is retired). Keep using `start-mcp-server` only if you need service-token auth or a self-hosted (BYOC) instance.
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

1. Ensure you have Go installed on your system (version 1.25 or later — see `go.mod`).
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
