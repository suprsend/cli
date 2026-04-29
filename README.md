# SuprSend CLI

[![cli MCP server](https://glama.ai/mcp/servers/suprsend/cli/badges/score.svg)](https://glama.ai/mcp/servers/suprsend/cli)

SuprSend CLI is a command-line interface tool for interacting with the SuprSend API, written in Go.

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

1. Ensure you have Go installed on your system (version 1.20 or later).
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

A CI check on PRs to `main` verifies that `docs/` and `skills/` are up to date. Run `make build` and commit the output before opening a PR.

### Removing local build artifacts
```bash
make clean
```

## License
This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
