import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { createPlugin } from "../index.js";

const packageURL = new URL("../package.json", import.meta.url);
const readmeURL = new URL("../../../README.md", import.meta.url);
const specURL = new URL("../../../openspec/changes/acm-opencode-claude-plugin/specs/acm-opencode-claude-auth/spec.md", import.meta.url);
const adrURL = new URL("../../../docs/03-architecture/decisions/0001-use-ecosystem-plugin-compatibility.md", import.meta.url);
const removedMatrix = /matriz fijada OpenCode 1\.18\.19 \/ SDK 1\.17\.12 \/ Claude CLI 2\.1\.236/;

function assertCompatibilityPolicy({ packageDocument, readme, spec, adr }) {
  assert.equal(packageDocument.dependencies["@opencode-ai/plugin"], "^1.18.18");
  assert.doesNotMatch(readme, removedMatrix);
  assert.match(readme, /rango declarado `@opencode-ai\/plugin: \^1\.18\.18`/);
  assert.match(readme, /detección de Claude CLI es solo diagnóstica/);
  assert.match(adr, /declare `@opencode-ai\/plugin` as `\^1\.18\.18`/);
  assert.match(adr, /Claude CLI detection as diagnostics only/);
  assert.match(adr, /superseding compatibility decision MUST amend its specification in the same slice/);
  assert.doesNotMatch(spec, /supported OpenCode\/Claude CLI matrix/);
  assert.match(spec, /declared `@opencode-ai\/plugin` package range SHALL govern plugin API compatibility/);
  assert.match(spec, /Claude CLI detection MUST be diagnostic-only; missing or non-exact CLI evidence MUST NOT block plugin load/);
}

async function loadWithClaudeEvidence(error, stdout) {
  const diagnostics = [];
  const plugin = createPlugin({
    platform: "linux",
    diagnostic: (event) => diagnostics.push(event),
    versionIO: { execFile: (_command, _args, _options, done) => done(error, stdout, "") },
  });
  return { hooks: await plugin(), diagnostics };
}

test("README, ADR, auth R3, and package agree with non-exact CLI evidence", async () => {
  const [packageText, readme, spec, adr] = await Promise.all([
    readFile(packageURL, "utf8"),
    readFile(readmeURL, "utf8"),
    readFile(specURL, "utf8"),
    readFile(adrURL, "utf8"),
  ]);
  const packageDocument = JSON.parse(packageText);
  const { hooks, diagnostics } = await loadWithClaudeEvidence(null, "9.9.9\n");

  assertCompatibilityPolicy({ packageDocument, readme, spec, adr });
  assert.deepEqual(Object.keys(hooks).sort(), ["auth", "chat.headers"]);
  assert.equal(diagnostics[0].version, "9.9.9");
});

test("missing CLI evidence cannot block load", async () => {
  const { hooks, diagnostics } = await loadWithClaudeEvidence(new Error("missing"), "");

  assert.equal(hooks.auth.provider, "anthropic");
  assert.equal(diagnostics[0].outcome, "unavailable");
});
