#!/usr/bin/env node
"use strict";

const { spawnSync } = require("node:child_process");
const fs = require("node:fs");
const path = require("node:path");

const isWin = process.platform === "win32";
const exeName = isWin ? "hamunaptra.exe" : "hamunaptra";
const localBin = path.join(__dirname, "..", ".bin", exeName);

if (!fs.existsSync(localBin)) {
  const hint = [
    "",
    "hamunaptra binary not found in npm package cache.",
    "Try reinstalling:",
    "  npm i -g hamunaptra-cli",
    "or run without global install:",
    "  npx -y hamunaptra-cli --help",
    "",
    "If your platform binary is not published yet, fallback to:",
    "  go install github.com/rodrigobatini/hamunaptra-cli/cmd/hamunaptra@latest",
    "",
  ].join("\n");
  process.stderr.write(hint);
  process.exit(1);
}

const child = spawnSync(localBin, process.argv.slice(2), {
  stdio: "inherit",
  env: process.env,
});

if (child.error) {
  process.stderr.write(`Failed to launch hamunaptra binary: ${child.error.message}\n`);
  process.exit(1);
}

process.exit(child.status == null ? 1 : child.status);

