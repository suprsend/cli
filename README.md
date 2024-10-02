# SuprSend CLI

SuprSend CLI is a command-line interface tool for interacting with the SuprSend API, written in Go.

## Installation

### Homebrew

You can install SuprSend CLI using Homebrew:

```bash
brew tap suprsend/tap
brew install suprsend
```

### Binary Releases

Pre-compiled binaries for various platforms are available on the [Releases page](https://github.com/suprsend/cli/releases).

### Building from Source

To build SuprSend CLI from source, follow these steps:

1. Ensure you have Go installed on your system (version 1.20 or later).
2. Clone the repository:
    
    ```bash
    git clone https://github.com/suprsend/cli.git
    cd cli
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

## Features

- Easy interaction with SuprSend API
- Cross-platform support (Windows, macOS, Linux)
- Automated releases for multiple architectures

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
