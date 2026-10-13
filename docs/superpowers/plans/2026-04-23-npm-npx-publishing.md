# npm / npx Publishing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship the SuprSend CLI to npm so users can run `npx suprsend` (or `npm i -g suprsend`) on macOS, Linux, and Windows (x64 + arm64).

**Architecture:** `optionalDependencies` pattern. Unscoped root package `suprsend` contains a tiny Node shim and declares six per-platform packages as optional deps. Each platform package (`@suprsend/cli-<os>-<arch>`) carries only the matching Go binary and `os`/`cpu` gates so npm installs exactly one. Package sources are checked-in templates under `npm/`. A bash script (`scripts/publish-npm.sh`) assembles the seven packages from `dist/` after goreleaser runs and publishes them. The existing `.github/workflows/release.yml` gains a Node step + publish step after the `Run GoReleaser` step.

**Tech Stack:** Node.js 20 (for the shim runtime and the publish step), bash (publish script), GitHub Actions, goreleaser (unchanged), npm.

**Design spec:** `docs/superpowers/specs/2026-04-23-npm-npx-publishing-design.md`

**Testing note:** This repo has no Go/JS test framework (by design, per CLAUDE.md). Verification uses runtime checks: execute the shim against a real goreleaser `dist/` on this machine, and dry-run the publish script end-to-end before wiring it into CI.

---

## Task 0: One-time prerequisites (done by a human, not the implementer)

**This task is not code. Mark it complete once the listed commands have been run by a human with access to the `@suprsend` npm org. The automated tasks below assume these are done.**

**Files:** none

- [ ] **Step 1: Claim the unscoped `suprsend` name with a placeholder version**

Do this on your workstation, logged in as a member of the `@suprsend` org (`npm whoami` → should be a member; `npm team ls suprsend` should list `@suprsend:developers`).

```bash
mkdir -p /tmp/suprsend-claim && cd /tmp/suprsend-claim
cat > package.json <<'JSON'
{
  "name": "suprsend",
  "version": "0.0.0",
  "description": "Reserved for the SuprSend CLI. See https://github.com/suprsend/cli",
  "license": "MIT",
  "repository": "github:suprsend/cli"
}
JSON
npm publish --access public
```

- [ ] **Step 2: Grant the org team ownership of the unscoped package**

```bash
npm access grant read-write suprsend:developers suprsend
npm access ls-collaborators suprsend   # verify @suprsend:developers is listed
```

- [ ] **Step 3: Create an org-level automation token and add it to GitHub Actions secrets**

1. https://www.npmjs.com/settings/suprsend/tokens → "Generate New Token" → type "Automation" (not Classic; Automation bypasses 2FA in CI).
2. Copy the token.
3. GitHub → `suprsend/cli` repo → Settings → Secrets and variables → Actions → New repository secret → name `NPM_TOKEN`, value = the token.

- [ ] **Step 4: Sanity-check the `@suprsend` scope**

```bash
npm view @suprsend 2>/dev/null || echo "no scoped pkgs yet (expected)"
npm org ls suprsend   # should list you as a member
```

No commit — this task touches no repo files.

---

## Task 1: Scaffold the `npm/` directory tree

**Files:**
- Create: `npm/suprsend/package.json`
- Create: `npm/suprsend/bin/suprsend.js`
- Create: `npm/suprsend/README.md`
- Create: `npm/platforms/darwin-x64/package.json`
- Create: `npm/platforms/darwin-x64/README.md`
- Create: `npm/platforms/darwin-arm64/package.json`
- Create: `npm/platforms/darwin-arm64/README.md`
- Create: `npm/platforms/linux-x64/package.json`
- Create: `npm/platforms/linux-x64/README.md`
- Create: `npm/platforms/linux-arm64/package.json`
- Create: `npm/platforms/linux-arm64/README.md`
- Create: `npm/platforms/win32-x64/package.json`
- Create: `npm/platforms/win32-x64/README.md`
- Create: `npm/platforms/win32-arm64/package.json`
- Create: `npm/platforms/win32-arm64/README.md`

- [ ] **Step 1: Create the root-package directories**

```bash
mkdir -p npm/suprsend/bin
mkdir -p npm/platforms/{darwin-x64,darwin-arm64,linux-x64,linux-arm64,win32-x64,win32-arm64}
```

- [ ] **Step 2: Write `npm/suprsend/package.json`**

Create `npm/suprsend/package.json`:

```json
{
  "name": "suprsend",
  "version": "0.0.0",
  "description": "SuprSend CLI — manage notification infrastructure from your terminal.",
  "bin": {
    "suprsend": "bin/suprsend.js"
  },
  "files": [
    "bin",
    "README.md",
    "LICENSE"
  ],
  "keywords": [
    "suprsend",
    "notifications",
    "cli",
    "workflow",
    "mcp"
  ],
  "repository": {
    "type": "git",
    "url": "https://github.com/suprsend/cli.git"
  },
  "homepage": "https://github.com/suprsend/cli",
  "bugs": {
    "url": "https://github.com/suprsend/cli/issues"
  },
  "license": "MIT",
  "engines": {
    "node": ">=18"
  },
  "optionalDependencies": {
    "@suprsend/cli-darwin-x64": "0.0.0",
    "@suprsend/cli-darwin-arm64": "0.0.0",
    "@suprsend/cli-linux-x64": "0.0.0",
    "@suprsend/cli-linux-arm64": "0.0.0",
    "@suprsend/cli-win32-x64": "0.0.0",
    "@suprsend/cli-win32-arm64": "0.0.0"
  }
}
```

The `0.0.0` version strings (both `version` and every `optionalDependencies` value) are placeholders. `scripts/publish-npm.sh` rewrites them to the release version at publish time.

- [ ] **Step 3: Write `npm/suprsend/bin/suprsend.js`**

Create `npm/suprsend/bin/suprsend.js`:

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

- [ ] **Step 4: Write `npm/suprsend/README.md`**

Create `npm/suprsend/README.md`:

````markdown
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
````

- [ ] **Step 5: Write all six platform `package.json` files**

Each file is identical in shape with the name / `os` / `cpu` varying. Create each file exactly as shown.

`npm/platforms/darwin-x64/package.json`:
```json
{
  "name": "@suprsend/cli-darwin-x64",
  "version": "0.0.0",
  "description": "SuprSend CLI binary for darwin-x64. Installed automatically by the `suprsend` package.",
  "os": ["darwin"],
  "cpu": ["x64"],
  "files": [
    "bin",
    "README.md",
    "LICENSE"
  ],
  "repository": {
    "type": "git",
    "url": "https://github.com/suprsend/cli.git"
  },
  "homepage": "https://github.com/suprsend/cli",
  "license": "MIT"
}
```

`npm/platforms/darwin-arm64/package.json`:
```json
{
  "name": "@suprsend/cli-darwin-arm64",
  "version": "0.0.0",
  "description": "SuprSend CLI binary for darwin-arm64. Installed automatically by the `suprsend` package.",
  "os": ["darwin"],
  "cpu": ["arm64"],
  "files": [
    "bin",
    "README.md",
    "LICENSE"
  ],
  "repository": {
    "type": "git",
    "url": "https://github.com/suprsend/cli.git"
  },
  "homepage": "https://github.com/suprsend/cli",
  "license": "MIT"
}
```

`npm/platforms/linux-x64/package.json`:
```json
{
  "name": "@suprsend/cli-linux-x64",
  "version": "0.0.0",
  "description": "SuprSend CLI binary for linux-x64. Installed automatically by the `suprsend` package.",
  "os": ["linux"],
  "cpu": ["x64"],
  "files": [
    "bin",
    "README.md",
    "LICENSE"
  ],
  "repository": {
    "type": "git",
    "url": "https://github.com/suprsend/cli.git"
  },
  "homepage": "https://github.com/suprsend/cli",
  "license": "MIT"
}
```

`npm/platforms/linux-arm64/package.json`:
```json
{
  "name": "@suprsend/cli-linux-arm64",
  "version": "0.0.0",
  "description": "SuprSend CLI binary for linux-arm64. Installed automatically by the `suprsend` package.",
  "os": ["linux"],
  "cpu": ["arm64"],
  "files": [
    "bin",
    "README.md",
    "LICENSE"
  ],
  "repository": {
    "type": "git",
    "url": "https://github.com/suprsend/cli.git"
  },
  "homepage": "https://github.com/suprsend/cli",
  "license": "MIT"
}
```

`npm/platforms/win32-x64/package.json`:
```json
{
  "name": "@suprsend/cli-win32-x64",
  "version": "0.0.0",
  "description": "SuprSend CLI binary for win32-x64. Installed automatically by the `suprsend` package.",
  "os": ["win32"],
  "cpu": ["x64"],
  "files": [
    "bin",
    "README.md",
    "LICENSE"
  ],
  "repository": {
    "type": "git",
    "url": "https://github.com/suprsend/cli.git"
  },
  "homepage": "https://github.com/suprsend/cli",
  "license": "MIT"
}
```

`npm/platforms/win32-arm64/package.json`:
```json
{
  "name": "@suprsend/cli-win32-arm64",
  "version": "0.0.0",
  "description": "SuprSend CLI binary for win32-arm64. Installed automatically by the `suprsend` package.",
  "os": ["win32"],
  "cpu": ["arm64"],
  "files": [
    "bin",
    "README.md",
    "LICENSE"
  ],
  "repository": {
    "type": "git",
    "url": "https://github.com/suprsend/cli.git"
  },
  "homepage": "https://github.com/suprsend/cli",
  "license": "MIT"
}
```

- [ ] **Step 6: Write platform README.md files**

All six platform `README.md` files have the same content, differing only in the platform name in the heading. Write each as:

`npm/platforms/darwin-x64/README.md`:
```markdown
# @suprsend/cli-darwin-x64

Platform-specific binary for the SuprSend CLI (`darwin-x64`).

You probably don't want to install this directly — install the [`suprsend`](https://www.npmjs.com/package/suprsend) package instead, which will pull this in automatically on matching machines.

## License

MIT
```

Repeat for `darwin-arm64`, `linux-x64`, `linux-arm64`, `win32-x64`, `win32-arm64`, changing only the heading (`# @suprsend/cli-<platform>`) and the text in parentheses after "SuprSend CLI".

- [ ] **Step 7: Verify file tree**

Run:
```bash
find npm -type f | sort
```

Expected output:
```
npm/platforms/darwin-arm64/README.md
npm/platforms/darwin-arm64/package.json
npm/platforms/darwin-x64/README.md
npm/platforms/darwin-x64/package.json
npm/platforms/linux-arm64/README.md
npm/platforms/linux-arm64/package.json
npm/platforms/linux-x64/README.md
npm/platforms/linux-x64/package.json
npm/platforms/win32-arm64/README.md
npm/platforms/win32-arm64/package.json
npm/platforms/win32-x64/README.md
npm/platforms/win32-x64/package.json
npm/suprsend/README.md
npm/suprsend/bin/suprsend.js
npm/suprsend/package.json
```

- [ ] **Step 8: JSON sanity-check the package.json files**

```bash
for f in npm/suprsend/package.json npm/platforms/*/package.json; do
  node -e "JSON.parse(require('fs').readFileSync('$f','utf8'))" && echo "ok: $f" || { echo "bad: $f"; exit 1; }
done
```

Expected: seven `ok:` lines, no errors.

- [ ] **Step 9: Commit**

```bash
git add npm/
git commit -m "feat(npm): scaffold package templates for npx publishing"
```

---

## Task 2: Write the publish script

**Files:**
- Create: `scripts/publish-npm.sh`

- [ ] **Step 1: Write `scripts/publish-npm.sh`**

Create `scripts/publish-npm.sh`:

```bash
#!/usr/bin/env bash
# Usage: scripts/publish-npm.sh <version>
#
# Assembles and publishes the suprsend npm packages from dist/ after goreleaser
# has run. Arg is the version to stamp (leading `v` is stripped).
#
# Env:
#   DRY_RUN=1    pass --dry-run to npm publish instead of really publishing.
#   NODE_AUTH_TOKEN / NPM_TOKEN   used by npm in CI. In a dry run, unset is fine.

set -euo pipefail

VERSION="${1:?usage: $0 <version>}"
VERSION="${VERSION#v}"

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIST="$ROOT/dist"
NPM_SRC="$ROOT/npm"
STAGE="$DIST/npm-stage"

if [[ ! -d "$DIST" ]]; then
  echo "error: $DIST does not exist. Run goreleaser first." >&2
  exit 1
fi

rm -rf "$STAGE"
mkdir -p "$STAGE"

# Per-platform metadata: <npm-arch> <go-os> <go-arch> <binary-filename>
PLATFORMS=(
  "darwin-x64   darwin  amd64  suprsend"
  "darwin-arm64 darwin  arm64  suprsend"
  "linux-x64    linux   amd64  suprsend"
  "linux-arm64  linux   arm64  suprsend"
  "win32-x64    windows amd64  suprsend.exe"
  "win32-arm64  windows arm64  suprsend.exe"
)

stamp_version() {
  # $1 = path to package.json, $2 = version, $3 = "root" or "platform"
  # For root packages, also rewrites every optionalDependencies value.
  node -e '
    const fs = require("fs");
    const path = process.argv[1];
    const v = process.argv[2];
    const role = process.argv[3];
    const p = JSON.parse(fs.readFileSync(path, "utf8"));
    p.version = v;
    if (role === "root" && p.optionalDependencies) {
      for (const k of Object.keys(p.optionalDependencies)) {
        p.optionalDependencies[k] = v;
      }
    }
    fs.writeFileSync(path, JSON.stringify(p, null, 2) + "\n");
  ' "$1" "$2" "$3"
}

# -------- Platform packages --------
for entry in "${PLATFORMS[@]}"; do
  read -r NPM_ARCH GO_OS GO_ARCH BIN_NAME <<<"$entry"
  NPM_PKG="@suprsend/cli-$NPM_ARCH"
  PKG_STAGE="$STAGE/$NPM_ARCH"

  mkdir -p "$PKG_STAGE/bin"
  cp "$NPM_SRC/platforms/$NPM_ARCH/package.json" "$PKG_STAGE/package.json"
  cp "$NPM_SRC/platforms/$NPM_ARCH/README.md"    "$PKG_STAGE/README.md"
  cp "$ROOT/LICENSE"                              "$PKG_STAGE/LICENSE"

  # goreleaser output dir has a go-version suffix (_v1, _v8.0, ...). Match with a glob.
  BIN_SRC_DIR=$(ls -d "$DIST/suprsend_${GO_OS}_${GO_ARCH}"_*/ 2>/dev/null | head -n1)
  if [[ -z "$BIN_SRC_DIR" ]]; then
    echo "error: no goreleaser output for ${GO_OS}_${GO_ARCH} under $DIST" >&2
    exit 1
  fi
  cp "${BIN_SRC_DIR%/}/$BIN_NAME" "$PKG_STAGE/bin/$BIN_NAME"
  if [[ "$BIN_NAME" != *.exe ]]; then
    chmod 0755 "$PKG_STAGE/bin/$BIN_NAME"
  fi

  stamp_version "$PKG_STAGE/package.json" "$VERSION" platform

  echo "==> Publishing $NPM_PKG@$VERSION"
  if [[ "${DRY_RUN:-}" == "1" ]]; then
    (cd "$PKG_STAGE" && npm publish --access public --dry-run)
  else
    if ! (cd "$PKG_STAGE" && npm publish --access public); then
      # Retry-friendly: if this exact version already exists, continue.
      if npm view "$NPM_PKG@$VERSION" version >/dev/null 2>&1; then
        echo "    $NPM_PKG@$VERSION already published, continuing."
      else
        echo "    failed to publish $NPM_PKG@$VERSION" >&2
        exit 1
      fi
    fi
  fi
done

# -------- Root package --------
ROOT_STAGE="$STAGE/suprsend"
mkdir -p "$ROOT_STAGE/bin"
cp "$NPM_SRC/suprsend/package.json"       "$ROOT_STAGE/package.json"
cp "$NPM_SRC/suprsend/bin/suprsend.js"    "$ROOT_STAGE/bin/suprsend.js"
cp "$NPM_SRC/suprsend/README.md"          "$ROOT_STAGE/README.md"
cp "$ROOT/LICENSE"                        "$ROOT_STAGE/LICENSE"
chmod 0755 "$ROOT_STAGE/bin/suprsend.js"

stamp_version "$ROOT_STAGE/package.json" "$VERSION" root

echo "==> Publishing suprsend@$VERSION"
if [[ "${DRY_RUN:-}" == "1" ]]; then
  (cd "$ROOT_STAGE" && npm publish --access public --dry-run)
else
  (cd "$ROOT_STAGE" && npm publish --access public)
fi

echo "Done. Published suprsend + 6 platform packages at $VERSION."
```

- [ ] **Step 2: Make it executable**

```bash
chmod +x scripts/publish-npm.sh
```

- [ ] **Step 3: Commit**

```bash
git add scripts/publish-npm.sh
git commit -m "feat(npm): add script to assemble and publish cli packages"
```

---

## Task 3: Verify the shim resolves and executes locally

**Files:** no new files; uses the existing `dist/` and `npm/` scaffolding.

This task confirms the shim actually launches the Go binary correctly on this machine before we wire anything into CI.

- [ ] **Step 1: Build a current `dist/` (if not already fresh)**

```bash
./scripts/build.sh snapshot
ls dist/suprsend_*_*_*/suprsend* 2>/dev/null | head
```

Expected: at least six binary paths printed (3 OSes × 2 arches — Windows ones are `suprsend.exe`).

- [ ] **Step 2: Assemble a local install tree by hand**

The point is to simulate what a user's `node_modules` looks like after `npm install suprsend`. We'll place the root package at `./local-test/node_modules/suprsend` and the single matching platform package at `./local-test/node_modules/@suprsend/cli-<host-platform>`.

```bash
rm -rf local-test && mkdir -p local-test/node_modules

# Root package.
cp -R npm/suprsend local-test/node_modules/suprsend

# Detect host platform and copy the matching platform package + binary.
HOST_PLATFORM=$(node -p "process.platform")
HOST_ARCH=$(node -p "process.arch")
NPM_ARCH="${HOST_PLATFORM}-${HOST_ARCH}"
GO_OS=$HOST_PLATFORM
case "$GO_OS" in
  win32) GO_OS=windows ;;
esac
GO_ARCH=$HOST_ARCH
case "$GO_ARCH" in
  x64) GO_ARCH=amd64 ;;
esac
BIN_NAME=suprsend
[[ "$HOST_PLATFORM" == "win32" ]] && BIN_NAME=suprsend.exe

mkdir -p "local-test/node_modules/@suprsend/cli-$NPM_ARCH/bin"
cp "npm/platforms/$NPM_ARCH/package.json" "local-test/node_modules/@suprsend/cli-$NPM_ARCH/package.json"
BIN_SRC=$(ls -d dist/suprsend_${GO_OS}_${GO_ARCH}_*/ | head -n1)
cp "${BIN_SRC%/}/$BIN_NAME" "local-test/node_modules/@suprsend/cli-$NPM_ARCH/bin/$BIN_NAME"
chmod 0755 "local-test/node_modules/@suprsend/cli-$NPM_ARCH/bin/$BIN_NAME"
```

- [ ] **Step 3: Run the shim**

```bash
node local-test/node_modules/suprsend/bin/suprsend.js --help
```

Expected: full `suprsend` help output — same thing `go run ./cmd/suprsend/main.go --help` prints.

- [ ] **Step 4: Confirm exit code propagation**

```bash
node local-test/node_modules/suprsend/bin/suprsend.js --definitely-not-a-real-flag
echo "exit=$?"
```

Expected: cobra prints an "unknown flag" error and `exit=` is non-zero (typically `1`).

- [ ] **Step 5: Confirm signal handling**

Pick any subcommand that blocks on input. `suprsend login` will usually block waiting for a service token or interactive prompt — start it and Ctrl-C it:

```bash
node local-test/node_modules/suprsend/bin/suprsend.js start-mcp-server &
PID=$!
sleep 1
kill -INT $PID
wait $PID
echo "exit=$?"
```

Expected: the Node process terminates promptly after the signal reaches the Go binary; `exit=` reflects a signal exit (usually `130` for SIGINT, i.e. `128 + 2`). This confirms the shim forwards signals correctly.

- [ ] **Step 6: Confirm the "missing platform package" error path**

Temporarily remove the platform package and retry:

```bash
mv "local-test/node_modules/@suprsend/cli-$NPM_ARCH" /tmp/pkg-backup
node local-test/node_modules/suprsend/bin/suprsend.js --help
echo "exit=$?"
mv /tmp/pkg-backup "local-test/node_modules/@suprsend/cli-$NPM_ARCH"
```

Expected: the "no prebuilt binary" error message from the shim, `exit=1`.

- [ ] **Step 7: Clean up**

```bash
rm -rf local-test
```

No commit — this task validates the shim without changing repo state.

---

## Task 4: Dry-run the publish script end-to-end

**Files:** no new files; validates `scripts/publish-npm.sh` against the existing `dist/`.

- [ ] **Step 1: Ensure `dist/` has the full matrix**

```bash
ls -d dist/suprsend_*_*_*/ | sort
```

Expected: six directories — `darwin_amd64_v1`, `darwin_arm64_v8.0`, `linux_amd64_v1`, `linux_arm64_v8.0`, `windows_amd64_v1`, `windows_arm64_v8.0` (suffixes may vary).

If it's empty, rerun `./scripts/build.sh snapshot` first.

- [ ] **Step 2: Run the publish script in DRY_RUN mode**

```bash
DRY_RUN=1 scripts/publish-npm.sh 9.9.9-dryrun
```

Expected output: seven `==> Publishing ...` lines (six platform packages, then the root), each followed by npm's `--dry-run` output listing the package contents and a "Tarball Contents" block. The script exits 0.

- [ ] **Step 3: Inspect the staged root package**

```bash
cat dist/npm-stage/suprsend/package.json
```

Expected: `"version": "9.9.9-dryrun"` at the top level, and every entry in `optionalDependencies` pinned to `"9.9.9-dryrun"`.

- [ ] **Step 4: Inspect one staged platform package**

```bash
cat dist/npm-stage/darwin-arm64/package.json
ls -la dist/npm-stage/darwin-arm64/bin/
file dist/npm-stage/darwin-arm64/bin/suprsend
```

Expected:
- `"version": "9.9.9-dryrun"`, `"os": ["darwin"]`, `"cpu": ["arm64"]`.
- `bin/suprsend` present, mode `-rwxr-xr-x`.
- `file` reports a Mach-O arm64 executable.

- [ ] **Step 5: Clean staging and commit**

```bash
rm -rf dist/npm-stage
```

No source files changed in this task — nothing to commit.

---

## Task 5: Extend the release workflow

**Files:**
- Modify: `.github/workflows/release.yml`

- [ ] **Step 1: Add Node setup and publish steps**

Open `.github/workflows/release.yml`. After the existing "Upload public key to release" step (currently ending with the `gh release upload` command and its `env:` block for `GH_TOKEN`), but **before** the "Clean up keys" step, insert the following two steps.

The current structure around lines 78–87:

```yaml
      - name: Upload public key to release
        run: gh release upload "${{ github.ref_name }}" /tmp/public_key.pem --clobber
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}

      - name: Clean up keys
        if: always()
        run: |
          rm -f /tmp/tap_key
          rm -f /tmp/public_key.pem
```

Becomes:

```yaml
      - name: Upload public key to release
        run: gh release upload "${{ github.ref_name }}" /tmp/public_key.pem --clobber
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}

      - name: Set up Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '20'
          registry-url: 'https://registry.npmjs.org'

      - name: Publish to npm
        env:
          NODE_AUTH_TOKEN: ${{ secrets.NPM_TOKEN }}
        run: scripts/publish-npm.sh "${GITHUB_REF_NAME#v}"

      - name: Clean up keys
        if: always()
        run: |
          rm -f /tmp/tap_key
          rm -f /tmp/public_key.pem
```

- [ ] **Step 2: Lint the YAML**

```bash
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/release.yml'))"
```

Expected: no output (silent success).

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/release.yml
git commit -m "ci: publish cli to npm after goreleaser"
```

---

## Task 6: Document `npx` / npm install in the project README

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Add an npm / npx section to the Installation block**

In `README.md`, find the current Installation section. It currently contains two subsections: `### Homebrew` and `### Binary Releases`. Add an `### npm / npx` subsection as the **first** installation option (before Homebrew), since `npx` is the lowest-friction path for new users.

Replace this block:

```markdown
## Installation

### Homebrew
```

with:

```markdown
## Installation

### npm / npx

Run without installing:

\`\`\`bash
npx suprsend --help
\`\`\`

Or install globally:

\`\`\`bash
npm i -g suprsend
suprsend --help
\`\`\`

Works on macOS, Linux, and Windows (x64 and arm64). Requires Node.js ≥ 18 (for `npx`).

### Homebrew
```

Note: the backticks in the example above are escaped with backslashes so this plan document renders cleanly. When editing `README.md`, use real backticks, not escaped ones.

- [ ] **Step 2: Visually verify**

```bash
head -40 README.md
```

Expected: new `### npm / npx` section appears before `### Homebrew`.

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "docs: document npx and npm installation for the cli"
```

---

## Task 7: Update CLAUDE.md with npm publishing context

**Files:**
- Modify: `CLAUDE.md`

Per the repo's pre-push convention, CLAUDE.md should reflect any major new release channel.

- [ ] **Step 1: Add a brief npm section under "Release"**

Open `CLAUDE.md`. Find the `### Release` subsection near the end. It currently reads:

```markdown
### Release

Uses goreleaser (`.goreleaser.yaml`). Builds include macOS notarization and Homebrew cask publishing to `suprsend/homebrew-tap`. The `make build` step compiles the `type-morph` Deno binary that gets embedded into the Go binary.
```

Replace with:

```markdown
### Release

Uses goreleaser (`.goreleaser.yaml`). Builds include macOS notarization and Homebrew cask publishing to `suprsend/homebrew-tap`. The `make build` step compiles the `type-morph` Deno binary that gets embedded into the Go binary.

After goreleaser, the release workflow also publishes to npm via `scripts/publish-npm.sh`. This ships seven packages at the same version: the unscoped root `suprsend` (a Node shim) and six platform packages `@suprsend/cli-<os>-<arch>` (darwin/linux/win32 × x64/arm64), each carrying the matching goreleaser binary. Package sources live under `npm/`. Users run `npx suprsend` or `npm i -g suprsend`.
```

- [ ] **Step 2: Commit**

```bash
git add CLAUDE.md
git commit -m "docs(claude): note npm publishing channel"
```

---

## Task 8: Final local verification and push

**Files:** none modified.

- [ ] **Step 1: Rerun the dry-run against a freshly rebuilt `dist/`**

```bash
./scripts/build.sh snapshot
DRY_RUN=1 scripts/publish-npm.sh 9.9.9-final-dry
```

Expected: seven `==> Publishing ...` lines, exit 0.

- [ ] **Step 2: Confirm the branch is clean**

```bash
git status
git log --oneline origin/main..HEAD
```

Expected: working tree clean; one commit per prior task (plus the spec commit from brainstorming).

- [ ] **Step 3: Push the branch**

```bash
git push -u origin npx-support
```

- [ ] **Step 4: Open a PR**

```bash
gh pr create --title "feat: publish cli to npm (npx suprsend)" --body "$(cat <<'EOF'
## Summary
- Publishes the CLI to npm via the `optionalDependencies` pattern so users can run `npx suprsend`.
- Adds `npm/` package templates (1 root + 6 per-platform) and `scripts/publish-npm.sh` to assemble + publish them from goreleaser output.
- Extends `.github/workflows/release.yml` with a Node setup + npm publish step after goreleaser.
- Documents `npx suprsend` in the project README.

## Prerequisites (one-time, done before merging)
- [ ] `suprsend` name claimed on npm (placeholder 0.0.0 published).
- [ ] `@suprsend:developers` granted read-write on the `suprsend` package.
- [ ] `NPM_TOKEN` (org Automation token) added to repo Actions secrets.

## Test plan
- [ ] Dry-run the publish script locally against a fresh `dist/` and confirm seven `==> Publishing` lines.
- [ ] Confirm the shim launches the Go binary on this host via the local node_modules simulation.
- [ ] After merge, cut a patch release tag and verify the npm registry shows all seven packages at the same version.
- [ ] Verify `npx suprsend --help` works on a clean machine (no prior install).
EOF
)"
```

---

## Self-Review (done before handing off)

**Spec coverage:**
- `optionalDependencies` root + 6 platform packages — Tasks 1, 2.
- Root shim via `require.resolve` + `spawnSync` — Task 1 Step 3; verified in Task 3 Steps 3–6.
- Checked-in `npm/` templates, dumb publish script — Tasks 1–2.
- GitHub Actions release workflow extension — Task 5.
- Version derived from git tag, minus leading `v` — `scripts/publish-npm.sh` handles it; Task 5 calls it with `${GITHUB_REF_NAME#v}`.
- Retry-friendly platform publish, strict root publish — script Step 1 handles both.
- Prerequisite human steps (name claim, org grant, `NPM_TOKEN`) — Task 0.
- README / CLAUDE.md updates — Tasks 6, 7.
- End-to-end local verification before push — Tasks 3, 4, 8.

**Placeholder scan:** No "TBD"/"TODO" instructions in any task. Every code block is complete and runnable. Version strings in templates are literal `0.0.0` by design and get rewritten by the publish script at run time.

**Type consistency:** All `@suprsend/cli-<os>-<arch>` names use the Node-vocabulary platform pairs (`darwin`/`linux`/`win32` × `x64`/`arm64`) consistently between the shim, the package templates, the publish script's `PLATFORMS` array, and the local-test setup in Task 3.

**Ambiguity check:** The goreleaser go-version suffix in output dir names (`_v1`, `_v8.0`) is explicitly handled by globbing — the plan doesn't assume fixed suffixes. The "claim the name" task is clearly flagged as a human prerequisite, not an automated step.
