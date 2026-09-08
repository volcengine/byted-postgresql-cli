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

import { execFileSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const dryRun = process.argv.includes("--dry-run");
const publicRegistry = "https://registry.npmjs.org/";
const registry = process.env.NPM_REGISTRY || publicRegistry;
const releaseBranch = process.env.RELEASE_BRANCH || "master";
const scope = "@byted-postgresql";
const mainPackage = `${scope}/cli`;
const binary = "byted-postgresql-cli";
const targets = [
  { goos: "darwin", goarch: "amd64", os: "darwin", cpu: "x64", ext: "" },
  { goos: "darwin", goarch: "arm64", os: "darwin", cpu: "arm64", ext: "" },
  { goos: "linux", goarch: "amd64", os: "linux", cpu: "x64", ext: "" },
  { goos: "linux", goarch: "arm64", os: "linux", cpu: "arm64", ext: "" },
  { goos: "windows", goarch: "amd64", os: "win32", cpu: "x64", ext: ".exe" },
  { goos: "windows", goarch: "arm64", os: "win32", cpu: "arm64", ext: ".exe" },
];

const capture = (command, args) =>
  execFileSync(command, args, { cwd: root, encoding: "utf8" }).trim();
const run = (command, args, options = {}) =>
  execFileSync(command, args, { cwd: root, stdio: "inherit", ...options });
const fail = (message) => {
  console.error(`\n${message}\n`);
  process.exit(1);
};
if (registry.replace(/\/+$/, "") !== publicRegistry.replace(/\/+$/, "")) {
  fail("publishing is restricted to the public npm registry: https://registry.npmjs.org/");
}

const branch = capture("git", ["rev-parse", "--abbrev-ref", "HEAD"]);
if (branch !== releaseBranch) fail(`release requires branch ${releaseBranch}; current branch is ${branch}`);
if (capture("git", ["status", "--porcelain"])) fail("release requires a clean working tree");

const tags = capture("git", ["tag", "--points-at", "HEAD"])
  .split("\n")
  .filter((tag) => /^v\d+\.\d+\.\d+$/.test(tag));
if (tags.length !== 1) fail("release requires exactly one vX.Y.Z tag at HEAD");
const version = tags[0].slice(1);
const gitHead = capture("git", ["rev-parse", "HEAD"]);
const modulePath = fs.readFileSync(path.join(root, "go.mod"), "utf8").match(/^module\s+(\S+)$/m)?.[1];
if (!modulePath) fail("go.mod has no module declaration");

const stage = path.join(root, "dist", "npm");
fs.rmSync(stage, { recursive: true, force: true });
fs.mkdirSync(stage, { recursive: true });
const platformPackage = (target) => `${scope}/cli-${target.os}-${target.cpu}`;
const common = {
  version,
  gitHead,
  description: "Volcengine PostgreSQL CLI native binary",
  license: "UNLICENSED",
  homepage: "https://github.com/volcengine/byted-postgresql-cli",
  repository: { type: "git", url: "https://github.com/volcengine/byted-postgresql-cli.git" },
  publishConfig: { registry },
};
const ldflags = `-s -w -X ${modulePath}/cmd.Version=${version}`;
const platformDirs = [];

for (const target of targets) {
  const name = platformPackage(target);
  const directory = path.join(stage, `cli-${target.os}-${target.cpu}`);
  fs.mkdirSync(path.join(directory, "bin"), { recursive: true });
  const output = path.join(directory, "bin", `${binary}${target.ext}`);
  run("go", ["build", "-trimpath", "-ldflags", ldflags, "-o", output, "."], {
    env: {
      ...process.env,
      CGO_ENABLED: "0",
      GOWORK: "off",
      GOOS: target.goos,
      GOARCH: target.goarch,
    },
  });
  fs.chmodSync(output, 0o755);
  fs.writeFileSync(
    path.join(directory, "package.json"),
    `${JSON.stringify({
      name,
      ...common,
      os: [target.os],
      cpu: [target.cpu],
      files: ["bin/"],
    }, null, 2)}\n`
  );
  platformDirs.push(directory);
}

const rootPackage = JSON.parse(fs.readFileSync(path.join(root, "package.json"), "utf8"));
const mainDir = path.join(stage, "cli");
fs.mkdirSync(path.join(mainDir, "bin"), { recursive: true });
for (const file of ["cli.js", "install.js", "paths.js"]) {
  fs.copyFileSync(path.join(root, "bin", file), path.join(mainDir, "bin", file));
}
fs.cpSync(path.join(root, "skills"), path.join(mainDir, "skills"), { recursive: true });
fs.writeFileSync(
  path.join(mainDir, "package.json"),
  `${JSON.stringify({
    ...rootPackage,
    version,
    gitHead,
    publishConfig: { registry },
    optionalDependencies: Object.fromEntries(targets.map((target) => [platformPackage(target), version])),
  }, null, 2)}\n`
);

const publishArgs = dryRun
  ? ["publish", "--dry-run", "--registry", registry]
  : ["publish", "--registry", registry];
for (const directory of platformDirs) run("npm", publishArgs, { cwd: directory });
run("npm", publishArgs, { cwd: mainDir });
console.log(`${dryRun ? "Dry-run assembled" : "Published"} ${mainPackage}@${version}`);
