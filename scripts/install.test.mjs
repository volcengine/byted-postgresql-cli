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

import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");

test("npm package exposes the Byte PostgreSQL launcher name", () => {
  const packageJSON = JSON.parse(fs.readFileSync(path.join(root, "package.json"), "utf8"));
  assert.equal(packageJSON.name, "@byted-postgresql/cli");
  assert.deepEqual(packageJSON.bin, { "byted-postgresql-cli": "bin/cli.js" });
  for (const dependency of Object.keys(packageJSON.optionalDependencies)) {
    assert.match(dependency, /^@byted-postgresql\/cli-/);
  }
});

test("launcher and installer use the new binary and data root", () => {
  const launcher = fs.readFileSync(path.join(root, "bin", "cli.js"), "utf8");
  const installer = fs.readFileSync(path.join(root, "bin", "install.js"), "utf8");
  const paths = fs.readFileSync(path.join(root, "bin", "paths.js"), "utf8");
  const combined = `${launcher}\n${installer}\n${paths}`;

  assert.match(combined, /byted-postgresql-cli/);
  assert.match(combined, /@byted-postgresql\/cli/);
  assert.match(combined, /\.volcengine-postgresql/);
  assert.doesNotMatch(combined, /bytedance-postgresql-cli/i);
  assert.doesNotMatch(combined, /bytecloud/i);
});

test("npm package includes the Volcengine PostgreSQL skill", () => {
  const skill = fs.readFileSync(path.join(root, "skills", "byted-postgresql", "SKILL.md"), "utf8");
  assert.match(skill, /name: byted-postgresql/);
  assert.match(skill, /Volcengine/);
  assert.doesNotMatch(skill, /BytePlus/);
  assert.match(skill, /MCP/);
  assert.ok(fs.existsSync(path.join(root, "skills", "byted-postgresql", "references", "volcengine.md")));
});
