import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { assertCompatibility, operationId, transformRequest } from "../compat.js";
import { createPlugin } from "../index.js";
import { refreshCredentials } from "../oauth.js";

const fixture = JSON.parse(await readFile(new URL("./fixtures/cases.json", import.meta.url)));

test("allows only Linux ACM profiles on the pinned compatibility matrix", () => {
  assert.doesNotThrow(() => assertCompatibility("linux", true, fixture.versions));
  assert.throws(() => assertCompatibility("darwin", true, fixture.versions), /unsupported/);
  assert.throws(() => assertCompatibility("linux", false, fixture.versions), /ACM-managed/);
  assert.throws(() => assertCompatibility("linux", true, { ...fixture.versions, sdk: "2.0.0" }), /unsupported/);
});

test("hashes stable OpenCode session and message identity", () => {
  assert.equal(operationId(fixture.hash.session, fixture.hash.message), fixture.hash.expected);
});

test("applies the pinned Claude request transform", () => {
  const output = transformRequest(fixture.request.input);
  assert.equal(output.system[0].text, fixture.request.system);
  assert.equal(output.tools[0].name, fixture.request.tool);
});

test("refreshes through ACM begin and commit without an auth-file write", async () => {
  const calls = [];
  const result = await refreshCredentials({ profile: "alpha", generation: 7 }, "b".repeat(64), fixture.credentials.source.claudeAiOauth, {
    machine: async (operation, body) => (calls.push([operation, body]), operation.endsWith("begin") ? { lease_id: "lease" } : { outcome: "committed" }),
    send: async () => new Response(JSON.stringify(fixture.credentials.response), { status: 200 }),
    now: () => 1000,
  });
  assert.equal(result.access, "new-access");
  assert.deepEqual(calls.map(([operation]) => operation), ["oauth.refresh.begin", "oauth.refresh.commit"]);
});

test("returns metadata for one OpenCode-owned attempt without replay or continuation hooks", async () => {
  let sends = 0;
  const plugin = await createPlugin({
    platform: "linux", versions: fixture.versions,
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
