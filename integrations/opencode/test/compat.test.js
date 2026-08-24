import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { assertCompatibility, detectClaudeVersion, operationId, transformRequest } from "../compat.js";
import { createPlugin } from "../index.js";
import { refreshCredentials } from "../oauth.js";

const fixture = JSON.parse(await readFile(new URL("./fixtures/cases.json", import.meta.url)));

function versionBoundary(output) {
  return { execFile(command, args, _options, callback) {
    assert.deepEqual([command, ...args], ["claude", "--version"]);
    callback(output === null ? new Error("unavailable") : null, output ?? "", "");
  } };
}

test("declares the developed-against OpenCode plugin API range", async () => {
  const metadata = JSON.parse(await readFile(new URL("../package.json", import.meta.url)));
  assert.equal(metadata.dependencies?.["@opencode-ai/plugin"], "^1.18.18");
});

test("keeps Linux and ACM profile state as hard preconditions", () => {
  assert.doesNotThrow(() => assertCompatibility("linux", true));
  assert.throws(() => assertCompatibility("darwin", true), /unsupported/);
  assert.throws(() => assertCompatibility("linux", false), /ACM-managed/);
});

test("loads without an exact runtime pin and records the Claude CLI diagnostic", async () => {
  const diagnostics = [];
  const plugin = await createPlugin({ platform: "linux", versionIO: versionBoundary("Claude Code 9.9.9\n"), diagnostic: async (event) => diagnostics.push(event) })();
  assert.deepEqual(Object.keys(plugin).sort(), ["auth", "chat.headers"]);
  assert.deepEqual(diagnostics, [{ component: "adapter", event: "compatibility", outcome: "recovered", retryable: false, version: "9.9.9" }]);
});

test("continues when the Claude CLI version cannot be resolved", async () => {
  const errors = [];
  const plugin = await createPlugin({ platform: "linux", versionIO: versionBoundary(null), diagnostic: async () => { throw new Error("diagnostics unavailable"); }, diagnosticError: (code) => errors.push(code) })();
  assert.deepEqual(Object.keys(plugin).sort(), ["auth", "chat.headers"]);
  assert.equal(await detectClaudeVersion(versionBoundary(null)), null);
  assert.deepEqual(errors, ["record_failed"]);
});

test("hashes stable OpenCode session and message identity", () => {
  assert.equal(operationId(fixture.hash.session, fixture.hash.message), fixture.hash.expected);
});

test("applies the Claude request transform", () => {
  const output = transformRequest(fixture.request.input);
  assert.equal(output.system[0].text, fixture.request.system);
  assert.equal(output.tools[0].name, fixture.request.tool);
});

test("refreshes through ACM begin and commit without an auth-file write", async () => {
  const calls = [];
  const result = await refreshCredentials({ profile: "alpha", generation: 7 }, "b".repeat(64), fixture.credentials.source.claudeAiOauth, {
    machine: async (operation, body) => (calls.push([operation, body]), operation.endsWith("begin") ? { lease_id: "lease" } : { outcome: "committed", generation: 8 }),
    send: async () => new Response(JSON.stringify(fixture.credentials.response), { status: 200 }),
    now: () => 1000,
  });
  assert.equal(result.access, "new-access");
  assert.equal(result.generation, 8);
  assert.deepEqual(calls.map(([operation]) => operation), ["oauth.refresh.begin", "oauth.refresh.commit"]);
});

test("returns metadata for one OpenCode-owned attempt without replay or continuation hooks", async () => {
  let sends = 0;
  const plugin = await createPlugin({
    platform: "linux", versionIO: versionBoundary("Claude Code 2.1.236\n"),
    machine: async () => ({ ok: true, profile: "alpha", config_dir: "/synthetic", generation: 1 }),
    read: async () => JSON.stringify({ claudeAiOauth: { accessToken: "access", refreshToken: "refresh", expiresAt: 9999999999999 } }),
    send: async (request) => (sends += 1, new Response(request.headers.get("x-acm-operation-id") ?? "clean", { status: 429 })),
  })({ client: { auth: { set: () => assert.fail("plugin must not write OpenCode auth") } } });
  const output = { headers: {} };
  await plugin["chat.headers"]({ sessionID: fixture.hash.session, message: { id: fixture.hash.message } }, output);
  const loaded = await plugin.auth.loader(async () => ({ type: "oauth" }), { models: {} });
  const response = await loaded.fetch(new Request("https://example.invalid", { headers: output.headers }));
  assert.equal(await response.text(), "clean");
  assert.equal(sends, 1);
  assert.deepEqual(Object.keys(plugin).sort(), ["auth", "chat.headers"]);
});
