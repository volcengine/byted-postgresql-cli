#!/usr/bin/env node
// Copyright (c) 2026 Beijing Volcano Engine Technology Co., Ltd.
// SPDX-License-Identifier: MIT
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.


"use strict";

import { spawnSync } from "node:child_process";
import { createRequire } from "node:module";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { CACHE_ROOT } from "./paths.js";

const require = createRequire(import.meta.url);
const args = process.argv.slice(2);
const self = fileURLToPath(import.meta.url);
const binDir = path.dirname(self);
const packageName = "@byted-postgresql/cli";
const platformPackage = `${packageName}-${process.platform}-${process.arch}`;
const binaryName = process.platform === "win32" ? "byted-postgresql-cli.exe" : "byted-postgresql-cli";
const nodeModules = process.platform === "win32" ? "node_modules" : path.join("lib", "node_modules");

function launcherVersion() {
  try {
    return JSON.parse(fs.readFileSync(path.join(binDir, "..", "package.json"), "utf8")).version;
  } catch {
    return "";
  }
}

function resolvedBinary() {
  const manifest = require.resolve(`${platformPackage}/package.json`);
  return path.join(path.dirname(manifest), "bin", binaryName);
}

function cachedBinary(version) {
  return path.join(CACHE_ROOT, version, nodeModules, ...platformPackage.split("/"), "bin", binaryName);
}

function installPlatformBinary(version) {
  const prefix = path.join(CACHE_ROOT, version);
  fs.mkdirSync(prefix, { recursive: true });
  const npmArgs = [
    "install",
    "-g",
    `${platformPackage}@${version}`,
    "--prefix",
    prefix,
    "--registry=https://registry.npmjs.org",
    `--os=${process.platform}`,
    `--cpu=${process.arch}`,
    "--ignore-scripts",
    "--no-audit",
    "--no-fund",
    "--loglevel=error",
  ];
  const result = spawnSync(process.platform === "win32" ? "npm.cmd" : "npm", npmArgs, {
    stdio: "inherit",
  });
  if (result.error || result.status !== 0) {
    throw result.error || new Error(`npm exited with status ${result.status}`);
  }
  return cachedBinary(version);
}

if (args[0] === "install" || args[0] === "uninstall") {
  const result = spawnSync(process.execPath, [path.join(binDir, "install.js"), ...args], {
    stdio: "inherit",
  });
  process.exit(typeof result.status === "number" ? result.status : 1);
}

const version = launcherVersion();
let binary = "";
try {
  binary = resolvedBinary();
} catch {
  if (version && fs.existsSync(cachedBinary(version))) {
    binary = cachedBinary(version);
  }
}

if (!binary && version && !process.env.VOLCENGINE_POSTGRESQL_CLI_NO_AUTO_DOWNLOAD) {
  try {
    binary = installPlatformBinary(version);
  } catch (error) {
    console.error(`[${packageName}] failed to install the native binary: ${error.message}`);
  }
}

if (!binary || !fs.existsSync(binary)) {
  console.error(
    `[${packageName}] native binary for ${process.platform}-${process.arch} was not found.\n` +
      `Run: npx --registry=https://registry.npmjs.org ${packageName}@latest install`
  );
  process.exit(1);
}

const result = spawnSync(binary, args, { stdio: "inherit" });
if (result.error) {
  console.error(`[${packageName}] ${result.error.message}`);
  process.exit(1);
}
process.exit(typeof result.status === "number" ? result.status : 1);
