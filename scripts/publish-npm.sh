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
