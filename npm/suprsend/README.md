# suprsend

SuprSend CLI — manage notification infrastructure from your terminal.

## Usage

```bash
npx suprsend --help
```

Or install globally:

```bash
npm i -g suprsend
suprsend --help
```

## How it works

This package installs a tiny Node launcher. On install, npm picks the matching prebuilt binary for your platform from one of these optional dependencies:

- `@suprsend/cli-darwin-x64`
- `@suprsend/cli-darwin-arm64`
- `@suprsend/cli-linux-x64`
- `@suprsend/cli-linux-arm64`
- `@suprsend/cli-win32-x64`
- `@suprsend/cli-win32-arm64`

If you install with `--no-optional` or `--omit=optional`, the binary won't be available. Reinstall without those flags to fix.

## Other install options

See the [project README](https://github.com/suprsend/cli#readme) for Homebrew and direct-binary options.

## License

MIT — see [LICENSE](./LICENSE).
