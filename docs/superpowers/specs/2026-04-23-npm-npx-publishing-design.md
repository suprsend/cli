# npm Publishing for the SuprSend CLI (`npx suprsend`)

**Date:** 2026-04-23
**Status:** Draft
**Branch:** `npx-support`

## Goal

Let users run the SuprSend CLI via `npx suprsend` (or install it globally with `npm i -g suprsend`). The experience should be:

```
$ npx suprsend --help
# ...suprsend help output...
```

No Go toolchain, no Homebrew, no manual binary download, no postinstall network call.

## Non-goals

- Rewriting the CLI in JavaScript. The Go binary stays the source of truth.
- Replacing Homebrew or GitHub Releases distribution — npm is an additional channel.
- Publishing the `gemini-cli-extension` binary to npm. That binary keeps its existing install path.
- Nightly or per-commit npm publishing. We publish on tagged releases only, matching the existing goreleaser flow.

## Strategy: `optionalDependencies` with per-platform packages

We adopt the pattern used by esbuild, @biomejs/biome, and swc: one small root package that users depend on, plus one package per supported platform that carries the matching native binary. npm's `os` / `cpu` fields gate which optional dependency actually installs on a given machine.

**Why not postinstall download:** requires network at install time, breaks offline installs, slows `npx` cold start, and adds a layer of failure modes (checksum drift, GitHub outage, proxy misconfig) we don't want in the install path.

**Why not bundle all binaries in one package:** ~100 MB of binaries for every install regardless of platform. Users and registries both suffer.

## Supported platforms

Matching the existing goreleaser matrix:

| npm package                      | OS      | Arch  | Binary source in `dist/`                           |
|----------------------------------|---------|-------|----------------------------------------------------|
| `@suprsend/cli-darwin-x64`       | macOS   | amd64 | `dist/suprsend_darwin_amd64_v1/suprsend`           |
| `@suprsend/cli-darwin-arm64`     | macOS   | arm64 | `dist/suprsend_darwin_arm64_v8.0/suprsend`         |
| `@suprsend/cli-linux-x64`        | Linux   | amd64 | `dist/suprsend_linux_amd64_v1/suprsend`            |
| `@suprsend/cli-linux-arm64`      | Linux   | arm64 | `dist/suprsend_linux_arm64_v8.0/suprsend`          |
| `@suprsend/cli-win32-x64`        | Windows | amd64 | `dist/suprsend_windows_amd64_v1/suprsend.exe`      |
| `@suprsend/cli-win32-arm64`      | Windows | arm64 | `dist/suprsend_windows_arm64_v8.0/suprsend.exe`    |

Names use Node's `process.platform` / `process.arch` vocabulary (`darwin`, `linux`, `win32`, `x64`, `arm64`) rather than Go's — the root shim reads those values directly to select a package.

The exact suffix on goreleaser output directories (`_v1`, `_v8.0`) is a goreleaser-version-compat marker and may change. The publish script globs `dist/suprsend_<os>_<arch>_*/suprsend[.exe]` rather than hardcoding suffixes.

## Package structure

### Root package: `suprsend`

```
suprsend/
├── package.json
├── bin/
│   └── suprsend.js        # Node shim, the entrypoint `bin` points at
├── README.md              # install + npx usage (short)
└── LICENSE
```

`package.json` shape (essentials):

```json
{
  "name": "suprsend",
  "version": "<stamped from tag>",
  "description": "SuprSend CLI - manage notification infrastructure from your terminal",
  "bin": { "suprsend": "bin/suprsend.js" },
  "files": ["bin", "README.md", "LICENSE"],
  "repository": "github:suprsend/cli",
  "homepage": "https://github.com/suprsend/cli",
  "license": "MIT",
  "optionalDependencies": {
    "@suprsend/cli-darwin-x64": "<same version>",
    "@suprsend/cli-darwin-arm64": "<same version>",
    "@suprsend/cli-linux-x64": "<same version>",
    "@suprsend/cli-linux-arm64": "<same version>",
    "@suprsend/cli-win32-x64": "<same version>",
    "@suprsend/cli-win32-arm64": "<same version>"
  }
}
```

Notes:
- All optional deps pinned to the exact same version as the root. `npm install` picks exactly the one matching the machine's `os`/`cpu`; the rest silently skip.
- `files` whitelist keeps the tarball to the shim + license + README only (~a few KB).
- No `dependencies`. No `postinstall`. No native build step.

### Root shim: `bin/suprsend.js`

```js
#!/usr/bin/env node
'use strict';

const { spawnSync } = require('node:child_process');
const { platform, arch } = process;
const pkg = `@suprsend/cli-${platform}-${arch}`;
const binName = platform === 'win32' ? 'suprsend.exe' : 'suprsend';

let binary;
try {
  binary = require.resolve(`${pkg}/bin/${binName}`);
} catch {
  console.error(
    `suprsend: no prebuilt binary for ${platform}-${arch}.\n` +
    `This platform isn't supported, or the optional dependency ${pkg} failed to install.\n` +
    `If you used --no-optional or --omit=optional, reinstall without those flags.`
  );
  process.exit(1);
}

const result = spawnSync(binary, process.argv.slice(2), { stdio: 'inherit' });
if (result.error) {
  console.error(`suprsend: failed to execute binary: ${result.error.message}`);
  process.exit(1);
}
if (result.signal) {
  process.kill(process.pid, result.signal);
}
process.exit(result.status ?? 1);
```

Why this shape:
- `spawnSync` — shim has nothing to do while the child runs. Sync simplifies signal handling: Ctrl-C from the terminal goes to both processes naturally, and we re-raise the terminating signal on ourselves before exit so shell semantics (`$?`, `$SIGNAL`) are preserved.
- `require.resolve` over hardcoded `node_modules/` traversal — works with pnpm, Yarn PnP, hoisting quirks, and monorepos.
- No dependencies on anything outside Node's standard library.
- ~25 lines. Nothing to break.

### Platform package: `@suprsend/cli-<os>-<arch>`

```
@suprsend/cli-darwin-arm64/
├── package.json
├── bin/
│   └── suprsend           # (or suprsend.exe for win32 packages)
├── README.md
└── LICENSE
```

`package.json` shape:

```json
{
  "name": "@suprsend/cli-darwin-arm64",
  "version": "<stamped from tag>",
  "description": "SuprSend CLI binary for darwin-arm64. Installed automatically by the `suprsend` package.",
  "os": ["darwin"],
  "cpu": ["arm64"],
  "files": ["bin", "README.md", "LICENSE"],
  "repository": "github:suprsend/cli",
  "license": "MIT"
}
```

Notes:
- `os` and `cpu` fields are what npm uses to skip incompatible optional deps. Getting these right is the whole point.
- No `bin` field — the root package owns the user-facing `suprsend` command. Platform packages are pure binary carriers.
- Binary is marked executable (`chmod 0755`) during packaging for macOS/Linux.

## Repo layout

```
npm/
├── suprsend/                       # root package, checked in mostly complete
│   ├── package.json                # `version` is a placeholder, stamped at publish time
│   ├── bin/suprsend.js
│   ├── README.md
│   └── LICENSE                     # copied from repo root
└── platforms/                      # per-platform package templates
    ├── darwin-x64/
    │   ├── package.json            # version placeholder; binary + README copied in at publish
    │   └── README.md
    ├── darwin-arm64/...
    ├── linux-x64/...
    ├── linux-arm64/...
    ├── win32-x64/...
    └── win32-arm64/...

scripts/
└── publish-npm.sh                  # assembles + publishes all 7 packages
```

**Why checked-in templates over generate-from-scratch:** Per-package `package.json` metadata (os/cpu, repo/license/keywords, descriptions) is non-trivial and worth reviewing in diffs. The publish script becomes a dumb assembler that substitutes only two things — version and binary — making the published output auditable from the repo.

## Release automation

### Trigger

Same as today: push a git tag → `.github/workflows/release.yml` runs. We extend that workflow with an npm publish step after goreleaser finishes.

### Workflow additions (sketch)

Appended to the existing release job, after the goreleaser step:

```yaml
- name: Set up Node.js
  uses: actions/setup-node@v4
  with:
    node-version: '20'
    registry-url: 'https://registry.npmjs.org'

- name: Publish to npm
  env:
    NODE_AUTH_TOKEN: ${{ secrets.NPM_TOKEN }}
  run: scripts/publish-npm.sh "${GITHUB_REF_NAME#v}"
```

The version argument is the git tag with any leading `v` stripped, matching the existing goreleaser version handling.

### Required secrets

- `NPM_TOKEN` — npm automation token with publish access to both the `suprsend` package and the `@suprsend` scope.

### `scripts/publish-npm.sh` behavior

Runs in the release workflow after `dist/` is populated by goreleaser. For each of the 6 platform packages:

1. Create a clean staging directory under `dist/npm-stage/<pkg>/`.
2. Copy `npm/platforms/<os>-<arch>/{package.json,README.md}` into it.
3. Copy the matching binary from `dist/suprsend_<os>_<arch>_*/suprsend[.exe]` into `<stage>/bin/`.
4. `chmod 0755` the binary (Unix platforms).
5. Stamp `version` in `package.json` (idempotent `sed` / `jq`).
6. `npm publish --access public` from the staging directory.

Then the root package:

1. Create `dist/npm-stage/suprsend/`.
2. Copy `npm/suprsend/*` into it.
3. Stamp `version` AND stamp all `optionalDependencies` values to the same version.
4. `npm publish --access public`.

**Ordering matters:** platform packages publish first so the root's optional deps resolve immediately on the registry. The script aborts the root publish if any platform publish failed.

**Idempotency / retry:** if the script re-runs for the same version, `npm publish` on a version that already exists returns a well-known error. The script treats "already exists" for platform packages as non-fatal (common in retry scenarios) and only fatal for the root package — so a partial-failure retry recovers cleanly.

### Versioning

- The version is always the git tag (leading `v` stripped). No separate npm version bump.
- All 7 packages publish together at the same version for every release.
- We do not publish snapshot builds to npm.

## Install UX

### `npx suprsend`
```
$ npx suprsend --help
```
First run downloads the root package + one platform package (~15–25 MB depending on platform). Subsequent `npx` runs hit the npx cache.

### Global install
```
$ npm i -g suprsend
$ suprsend --help
```

### Project-local install
```
$ npm i -D suprsend
$ npx suprsend workflow list
```

### Error paths

- **Unsupported platform** (e.g., linux/ppc64): the shim prints a clear message and exits 1. `npm install` itself succeeds — no optional dep gets installed, and the user only discovers the unsupported state when they run `suprsend`.
- **`--omit=optional` / `--no-optional`**: same failure mode, and the error message names this explicitly so the user knows the fix.

## Risk & open items

- **npm name squatting:** `suprsend` currently 404s on npm (confirmed). We should publish a `0.0.0` placeholder version from this branch before opening the PR to claim the name.
- **Binary size vs registry limits:** each platform package is ~15–25 MB. npm's per-package limit is 250 MB, so we're comfortably under.
- **macOS universal binary:** goreleaser produces a darwin universal binary alongside per-arch binaries. We use the per-arch binaries for symmetry with linux/windows. If goreleaser's `universal_binaries: replace: true` ever actually deletes the per-arch outputs in a future version, we fall back to using the universal binary for both darwin packages (`cpu: ['x64']` and `cpu: ['arm64']` pointing at the same physical binary). Keep an eye on it but no action needed now.
- **macOS code signing / notarization:** the per-arch binaries goreleaser emits are the ones that get signed and notarized (same flow as today). The shim invokes them directly, so notarization protection carries through. Gatekeeper should accept them on end-user machines the same way it does for the Homebrew cask.
- **CI token scope:** `NPM_TOKEN` needs publish rights for both `suprsend` (unscoped) and `@suprsend/*` (scoped). An org-level automation token is the cleanest fit.

## What we're not doing (explicitly)

- No postinstall download fallback. If a user's platform is unsupported, they get a clear error, not a surprise curl.
- No auto-update in the JS wrapper. Users update by bumping the npm version like any other dep.
- No separate "stable" vs "latest" dist-tags yet. Every release publishes to `latest`. If we need a `next` channel later, it's a one-line addition.
- No changes to the Homebrew, GitHub Release, or Gemini extension flows. They keep working exactly as they do today.
