import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { createPlugin } from "../index.js";

const packageURL = new URL("../package.json", import.meta.url);
const readmeURL = new URL("../../../README.md", import.meta.url);
const specURL = new URL("../../../openspec/specs/acm-opencode-claude-auth/spec.md", import.meta.url);
const failoverSpecURL = new URL("../../../openspec/specs/acm-opencode-claude-failover/spec.md", import.meta.url);
const designURL = new URL("../../../openspec/changes/archive/2026-08-24-acm-opencode-claude-plugin/design.md", import.meta.url);
const adrURL = new URL("../../../docs/03-architecture/decisions/0001-use-ecosystem-plugin-compatibility.md", import.meta.url);
const removedMatrix = /matriz fijada OpenCode 1\.18\.19 \/ SDK 1\.17\.12 \/ Claude CLI 2\.1\.236/;
const replayException = "ACM MAY accept a stale submitted generation only for a same-operation, same-profile ledger replay; it MUST return the current generation and MUST return before state persistence.";

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

function assertReplayExceptionPolicy(failoverSpec, design) {
  for (const document of [failoverSpec, design]) {
    assert.match(document, /Stale non-replay transitions MUST be rejected\./);
    assert.equal(document.includes(replayException), true);
  }
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

test("authoritative documents agree on compatibility and replay exceptions", async () => {
  const [packageText, readme, spec, failoverSpec, design, adr] = await Promise.all([
    readFile(packageURL, "utf8"),
    readFile(readmeURL, "utf8"),
    readFile(specURL, "utf8"),
    readFile(failoverSpecURL, "utf8"),
    readFile(designURL, "utf8"),
    readFile(adrURL, "utf8"),
  ]);
  const packageDocument = JSON.parse(packageText);
  const { hooks, diagnostics } = await loadWithClaudeEvidence(null, "9.9.9\n");

  assertCompatibilityPolicy({ packageDocument, readme, spec, adr });
  assertReplayExceptionPolicy(failoverSpec, design);
  assert.deepEqual(Object.keys(hooks).sort(), ["auth", "chat.headers"]);
  assert.equal(diagnostics[0].version, "9.9.9");
});

test("missing CLI evidence cannot block load", async () => {
  const { hooks, diagnostics } = await loadWithClaudeEvidence(new Error("missing"), "");

  assert.equal(hooks.auth.provider, "anthropic");
  assert.equal(diagnostics[0].outcome, "unavailable");
});
