#!/usr/bin/env node
import { platform, arch } from "node:os";
import { createRequire } from "node:module";
import { chmodSync, accessSync, constants } from "node:fs";
import { spawn } from "node:child_process";

const require = createRequire(import.meta.url);
const supportedTargets = new Set([
  "linux-x64",
  "linux-arm64",
  "darwin-x64",
  "darwin-arm64",
  "win32-x64",
  "win32-arm64",
]);

const target = `${platform()}-${arch()}`

if (!supportedTargets.has(target)) {
  console.error(`asimonim: Unsupported platform/arch: ${target}`);
  process.exit(1);
}

let binPath;
try {
  binPath = require.resolve(`@pwrs/asimonim-${target}/asimonim${platform() === 'win32' ? '.exe' : ''}`);
} catch {
  console.error(`asimonim: Platform binary package @pwrs/asimonim-${target} not installed. Was there an install error?`);
  process.exit(1);
}

// npm tarball extraction can strip the execute bit; fix it before spawning
if (platform() !== 'win32') {
  try {
    accessSync(binPath, constants.X_OK);
  } catch {
    chmodSync(binPath, 0o755);
  }
}

const child = spawn(binPath, process.argv.slice(2), { stdio: "inherit" });

// Forward common termination signals to child
const signals = ['SIGTERM', 'SIGINT', 'SIGHUP'];
signals.forEach(signal => {
  process.on(signal, () => {
    child.kill(signal);
  });
});

child.on('error', (err) => {
  console.error(`asimonim: Failed to spawn binary: ${err.message}`);
  process.exit(1);
});

child.on('exit', (code, signal) => {
  if (signal) {
    const signum = { SIGHUP: 1, SIGINT: 2, SIGTERM: 15 }[signal] ?? 1;
    process.exit(128 + signum);
  }
  process.exit(code ?? 1);
});
