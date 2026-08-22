import assert from "node:assert/strict";
import { mkdtemp, mkdir, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { spawnSync } from "node:child_process";
import test from "node:test";
import { fileURLToPath } from "node:url";

import { createPlugin } from "../index.js";
import { runMachine } from "../machine.js";

const fixture = JSON.parse(await readFile(new URL("./fixtures/quota.json", import.meta.url)));
const repository = fileURLToPath(new URL("../../..", import.meta.url));
test("maps real machine transitions without owning retry or continuation", async (t) => {
  const root = await mkdtemp(join(tmpdir(), "acm-quota-integration-"));
  t.after(() => rm(root, { recursive: true, force: true }));
  const binary = join(root, "acm");
  const acmDir = join(root, "acm-state");
  const stateDir = join(acmDir, "state");
  const statePath = join(stateDir, "opencode-machine-v1.json");
  const opencodeRoot = join(root, "opencode");
  const sdkRoot = join(opencodeRoot, "node_modules", "@opencode-ai", "sdk");
  const env = { ...process.env, HOME: join(root, "home"), ACM_DIR: acmDir,
    ACM_OPENCODE_CONFIG_HOME: join(root, "opencode-config"), ACM_DEFAULT_COOLDOWN_MIN: "1" };
  const build = spawnSync("go", ["build", "-o", binary, "."], { cwd: repository, encoding: "utf8" });
  assert.equal(build.status, 0, build.stderr);
  await mkdir(sdkRoot, { recursive: true });
  await mkdir(stateDir, { recursive: true });
  await writeFile(join(opencodeRoot, "package.json"), JSON.stringify({ name: "opencode-ai", version: "1.18.19" }));
  await writeFile(join(sdkRoot, "package.json"), JSON.stringify({ name: "@opencode-ai/sdk", version: "1.17.12" }));
  const versionIO = { execFile(command, _args, _options, callback) {
    callback(null, command === "which" ? `${join(opencodeRoot, "bin", "opencode")}\n` : "2.1.236 (Claude Code)\n", "");
  } };
  const credential = (expiresAt) => JSON.stringify({ claudeAiOauth: {
    accessToken: "synthetic-access", refreshToken: "synthetic-refresh", expiresAt,
  } });
  for (const profile of ["alpha", "beta"]) {
    const dir = join(acmDir, "profiles", "claude", profile);
    await mkdir(dir, { recursive: true });
    await writeFile(join(dir, ".credentials.json"), credential(9999999999999), { mode: 0o600 });
  }
  const machineCalls = [];
  const machine = async (operation, fields) => {
    const response = await runMachine(operation, fields, { binary, env });
    machineCalls.push([operation, fields, response]);
    return response;
  };
  const attempt = async (session, now, send = async () => assert.fail("provider must not be called")) => {
    const plugin = await createPlugin({ platform: "linux", versionIO, machine, now, send })();
    const output = { headers: {} };
    await plugin["chat.headers"]({ sessionID: session, message: { id: "message" } }, output);
    const auth = await plugin.auth.loader(async () => ({ type: "oauth" }));
    const response = await auth.fetch(new Request("https://example.invalid", { headers: output.headers }));
    return { operationID: output.headers["x-acm-operation-id"], plugin, response };
  };
  const epoch = Math.floor(Date.now() / 1000);

  await writeFile(statePath, JSON.stringify({ generation: 4, operations: [], cooling: { alpha: epoch + 90, beta: epoch + 120 } }));
  const cooling = await attempt("cooling", () => epoch * 1000);
  assert.equal(cooling.response.status, 429);
  assert.equal(cooling.response.headers.get("retry-after"), "90");
  t.diagnostic(`real binary cooling mapping: status=${cooling.response.status} headers=${JSON.stringify(Object.fromEntries(cooling.response.headers))}`);

  await writeFile(statePath, JSON.stringify({ generation: 4, operations: [], quarantined: ["alpha", "beta"] }));
  const quarantined = await attempt("quarantined", () => epoch * 1000);
  assert.equal(quarantined.response.status, 401);
  assert.deepEqual(await quarantined.response.json(), { action: "acm login", outcome: "quarantined", retryable: false });
  t.diagnostic(`real binary quarantined mapping: status=${quarantined.response.status} headers=${JSON.stringify(Object.fromEntries(quarantined.response.headers))}`);

  await writeFile(statePath, JSON.stringify({ generation: 4, operations: [], quarantined: ["alpha"], cooling: { alpha: epoch + 30, beta: epoch + 150 } }));
  const mixed = await attempt("mixed", () => epoch * 1000);
  assert.equal(mixed.response.headers.get("retry-after"), "150");

  await writeFile(statePath, JSON.stringify({ generation: 0, operations: [] }));
  await writeFile(join(acmDir, "profiles", "claude", "alpha", ".credentials.json"), credential(1), { mode: 0o600 });
  machineCalls.length = 0;
  let providerCalls = 0;
  const refreshed = await attempt("refresh-quota", () => epoch * 1000, async (input) => {
    if (typeof input === "string") return Response.json({ access_token: "new-access", refresh_token: "new-refresh", expires_in: 3600 });
    providerCalls += 1;
    return responseFor(fixture.confirmed);
  });
  const quota = machineCalls.find(([operation]) => operation === "quota.exhaust");
  const diagnostic = machineCalls.find(([operation]) => operation === "diagnostics.record");
  assert.equal(providerCalls, 1);
  assert.equal(quota[1].generation, 2);
  assert.equal(quota[2].generation, 3);
  assert.equal(refreshed.response.headers.get("retry-after"), null);
  t.diagnostic(`real binary replacement mapping: status=${refreshed.response.status} headers=${JSON.stringify(Object.fromEntries(refreshed.response.headers))}`);
  assert.deepEqual(diagnostic[1], { operation_id: diagnostic[1].operation_id, component: "quota", event: "transition", outcome: "cooling", retryable: true });
  const replacement = await runMachine("credential.select", { operation_id: refreshed.operationID }, { binary, env });
  assert.equal(replacement.profile, "beta");
  assert.deepEqual(Object.keys(refreshed.plugin).sort(), ["auth", "chat.headers"]);
  t.diagnostic(`real binary refresh/quota rotation: request-generation=${quota[1].generation} response-generation=${quota[2].generation} replacement=${replacement.profile}`);

  for (const status of [401, 429, 529]) {
    await writeFile(statePath, JSON.stringify({ generation: 0, operations: [] }));
    const original = Response.json({ error: { type: "overloaded_error" } }, { status });
    const passthrough = await attempt(`passthrough-${status}`, () => epoch * 1000, async () => original);
    assert.equal(passthrough.response, original);
    assert.equal(passthrough.response.headers.get("retry-after"), null);
    t.diagnostic(`real binary unconfirmed passthrough: status=${passthrough.response.status} headers=${JSON.stringify(Object.fromEntries(passthrough.response.headers))}`);
  }

  const blocked = join(root, "blocked-acm-dir");
  await writeFile(blocked, "not a directory");
  env.ACM_DIR = blocked;
  const unavailable = await attempt("unavailable", () => epoch * 1000);
  assert.equal(unavailable.response.status, 503);
  assert.deepEqual(await unavailable.response.json(), {
    code: "state_unavailable", outcome: "unavailable", retryable: false,
  });
});

function responseFor(value) {
  return new Response(JSON.stringify(value.body), {
    status: value.status,
    headers: value.headers,
  });
}
