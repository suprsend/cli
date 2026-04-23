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
