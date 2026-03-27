#!/usr/bin/env node
"use strict";

const fs = require("node:fs");
const path = require("node:path");
const https = require("node:https");

const pkg = require("../package.json");
const repo = "rodrigobatini/hamunaptra-cli";

const TARGETS = {
  "linux:x64": { suffix: "linux-amd64", ext: "" },
  "linux:arm64": { suffix: "linux-arm64", ext: "" },
  "darwin:x64": { suffix: "darwin-amd64", ext: "" },
  "darwin:arm64": { suffix: "darwin-arm64", ext: "" },
  "win32:x64": { suffix: "windows-amd64", ext: ".exe" },
  "win32:arm64": { suffix: "windows-arm64", ext: ".exe" },
};

function getTarget() {
  const key = `${process.platform}:${process.arch}`;
  return TARGETS[key] || null;
}

function ensureDir(dir) {
  fs.mkdirSync(dir, { recursive: true });
}

function downloadToFile(url, filePath) {
  return new Promise((resolve, reject) => {
    const out = fs.createWriteStream(filePath);
    https
      .get(url, (res) => {
        if (res.statusCode && res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          out.close();
          fs.rmSync(filePath, { force: true });
          downloadToFile(res.headers.location, filePath).then(resolve).catch(reject);
          return;
        }
        if (res.statusCode !== 200) {
          out.close();
          fs.rmSync(filePath, { force: true });
          reject(new Error(`HTTP ${res.statusCode} while downloading ${url}`));
          return;
        }
        res.pipe(out);
        out.on("finish", () => out.close(resolve));
      })
      .on("error", (err) => {
        out.close();
        fs.rmSync(filePath, { force: true });
        reject(err);
      });
  });
}

async function resolveVersionTag() {
  if (process.env.HAMUNAPTRA_CLI_VERSION_TAG) return process.env.HAMUNAPTRA_CLI_VERSION_TAG;
  const requested = process.env.npm_package_version || pkg.version;
  const candidate = requested.startsWith("v") ? requested : `v${requested}`;
  return candidate;
}

async function main() {
  if (process.env.HAMUNAPTRA_SKIP_POSTINSTALL === "1") return;

  const target = getTarget();
  if (!target) {
    console.warn(`[hamunaptra-cli] Unsupported platform ${process.platform}/${process.arch}; skipping binary download.`);
    return;
  }

  const tag = await resolveVersionTag();
  const asset = `hamunaptra-${target.suffix}${target.ext}`;
  const downloadUrl =
    process.env.HAMUNAPTRA_CLI_BINARY_URL ||
    `https://github.com/${repo}/releases/download/${tag}/${asset}`;

  const binDir = path.join(__dirname, "..", ".bin");
  const outPath = path.join(binDir, `hamunaptra${target.ext}`);
  ensureDir(binDir);

  try {
    await downloadToFile(downloadUrl, outPath);
    if (process.platform !== "win32") {
      fs.chmodSync(outPath, 0o755);
    }
    console.log(`[hamunaptra-cli] Installed ${asset}`);
  } catch (err) {
    fs.rmSync(outPath, { force: true });
    console.warn("[hamunaptra-cli] Could not download prebuilt binary.");
    console.warn(`[hamunaptra-cli] Attempted: ${downloadUrl}`);
    console.warn(`[hamunaptra-cli] Reason: ${err.message}`);
    console.warn("[hamunaptra-cli] You can still install from source:");
    console.warn("  go install github.com/rodrigobatini/hamunaptra-cli/cmd/hamunaptra@latest");
  }
}

main().catch((err) => {
  console.warn(`[hamunaptra-cli] postinstall failed: ${err.message}`);
});

