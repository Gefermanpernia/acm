import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { createPlugin } from "../index.js";

const packageURL = new URL("../package.json", import.meta.url);
const specURL = new URL("../../../openspec/changes/acm-opencode-claude-plugin/specs/acm-opencode-claude-auth/spec.md", import.meta.url);
const adrURL = new URL("../../../docs/03-architecture/decisions/0001-use-ecosystem-plugin-compatibility.md", import.meta.url);

async function loadWithClaudeEvidence(error, stdout) {
  const diagnostics = [];
  const plugin = createPlugin({
    platform: "linux",
    diagnostic: (event) => diagnostics.push(event),
    versionIO: { execFile: (_command, _args, _options, done) => done(error, stdout, "") },
  });
  return { hooks: await plugin(), diagnostics };
}

test("package-range compatibility and non-exact CLI evidence agree with auth R3", async () => {
  const packageDocument = JSON.parse(await readFile(packageURL, "utf8"));
  const spec = await readFile(specURL, "utf8");
  const { hooks, diagnostics } = await loadWithClaudeEvidence(null, "9.9.9\n");

  assert.equal(packageDocument.dependencies["@opencode-ai/plugin"], "^1.18.18");
  assert.deepEqual(Object.keys(hooks).sort(), ["auth", "chat.headers"]);
  assert.equal(diagnostics[0].version, "9.9.9");
  assert.doesNotMatch(spec, /supported OpenCode\/Claude CLI matrix/);
  assert.match(spec, /Claude CLI detection MUST be diagnostic-only/);
});

test("missing CLI evidence cannot block load and superseding decisions amend the spec", async () => {
  const adr = await readFile(adrURL, "utf8");
  const { hooks, diagnostics } = await loadWithClaudeEvidence(new Error("missing"), "");

  assert.equal(hooks.auth.provider, "anthropic");
  assert.equal(diagnostics[0].outcome, "unavailable");
  assert.match(adr, /superseding compatibility decision MUST amend its specification in the same slice/);
});
